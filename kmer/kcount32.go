package kmer

import (
	"bufio"
	"os"
	"sort"
	"strconv"
)

type Kcount32 struct {
	K   int        // Value of K
	Id  int        // Counter ID
	Con []uint32   // Byte to base convertion
	Fwd int        // Forward word operation
	Bwd int        // Backward word operation
	Wrd []uint32   // Word ID
	Val [][]uint32 // Counted values
	Nam []string   // Lib name
	Std int        // Number of distinct words
	Ust bool       // Unstranded Kmer count
}

func NewKcount32(id int, k int, ust bool) *Kcount32 {
	var c Kcount32
	c.K = k
	c.Id = id
	c.Con = make([]uint32, 256)
	c.Fwd = (16 - k + 1) * 2
	c.Bwd = (16 - k) * 2
	c.Val = make([][]uint32, 1)
	c.Nam = make([]string, 1)
	c.Ust = ust

	// setup base convertion (merge upper and lower cases)
	c.Con['C'] = uint32(1)
	c.Con['c'] = uint32(1)
	c.Con['G'] = uint32(2)
	c.Con['g'] = uint32(2)
	c.Con['T'] = uint32(3)
	c.Con['T'] = uint32(3)

	return &c
}

type Kcounts32 struct {
	Cou []*Kcount32
}

func NewKcounts32(th int, k int, ust bool) *Kcounts32 {
	var cs Kcounts32
	cs.Cou = make([]*Kcount32, th)
	for i := 0; i < th; i++ {
		cs.Cou[i] = NewKcount32(i, k, ust)
	}
	return &cs
}

/*
	Kcount32 methods
*/
// Set counter name
func (c *Kcount32) SetName(s string, i int) {
	if i >= len(c.Nam) {
		panic("Name index to large, you must extend the counter first.")
	}
	c.Nam[i] = s
}

// Count words from sequences of bytes provided by a channel
func (c *Kcount32) Count(seqChan chan []byte, couChan chan int) {
	// First, enumarate all words in a list
	rawList := make([]uint32, 0)
	nw := 0 // Number of words in the rawList

	// Retrieve sequence in the seq channel
	for seq := range seqChan {
		l := len(seq)

		// Extend the rawlist
		rawList = append(rawList, make([]uint32, l-c.K+1)...)

		// Init the first word
		w := uint32(0)
		for i := 0; i < c.K; i++ {
			w = (w << 2) | c.Con[seq[i]]
		}

		// Add the first word in the raw list of words
		rawList[nw] = w
		nw++

		// Continue to enumerate words
		for i := c.K; i < l; i++ {
			w = (w<<c.Fwd)>>c.Bwd | c.Con[seq[i]]
			rawList[nw] = w
			nw++
		}
	}

	// If unstranded count, replace words before sorting
	if c.Ust {
		km := NewKmer32(c.K)
		i := 0
		for i < nw {
			irc := km.Kmer32RevComp(rawList[i])
			if irc < rawList[i] {
				// Replace the word
				rawList[i] = irc
			}
			i++
		}
	}

	// Sort the rawList
	sort.Slice(rawList, func(i, j int) bool {
		return rawList[i] < rawList[j]
	})

	// Count words
	c.Wrd = make([]uint32, nw)
	c.Val[0] = make([]uint32, nw)
	i := 0
	stored := 0
	for i < nw {
		c.Wrd[stored] = rawList[i]
		val := 1
		j := i + 1
		for j < nw && rawList[i] == rawList[j] {
			val++
			j++
		}
		c.Val[0][stored] = uint32(val)
		i += val
		stored++
	}
	c.Std = stored

	// Though counter ID the couting channel
	couChan <- c.Id
}

// Write the Kmer counter into a file
func (c *Kcount32) Write(output string) {
	// Create the file handle
	f, e := os.Create(output)
	if e != nil {
		panic(e)
	}
	defer f.Close()
	b := bufio.NewWriter(f)

	// Write out the header
	b.WriteString("Kmer")
	for i := 0; i < len(c.Nam); i++ {
		b.WriteByte('\t')
		b.WriteString(c.Nam[i])
	}
	b.WriteByte('\n')

	// Init. a Kmer32 manager
	km := NewKmer32(c.K)

	nc := len(c.Val)
	for i := 0; i < c.Std; i++ {
		wb := km.Kmer32ToBytes(c.Wrd[i])
		b.Write(wb)
		for j := 0; j < nc; j++ {
			b.WriteByte('\t')
			b.WriteString(strconv.FormatUint(uint64(c.Val[j][i]), 10))
		}
		b.WriteByte('\n')
	}
	b.Flush()
}

func (c *Kcount32) WriteAll(output string) {
	// Create the file handle
	f, e := os.Create(output)
	if e != nil {
		panic(e)
	}
	defer f.Close()
	b := bufio.NewWriter(f)

	// Write out the header
	b.WriteString("Kmer")
	for i := 0; i < len(c.Nam); i++ {
		b.WriteByte('\t')
		b.WriteString(c.Nam[i])
	}
	b.WriteByte('\n')

	// Init. a Kmer32 manager
	km := NewKmer32(c.K)

	nc := len(c.Val)
	for i := 0; i < c.Std; i++ {
		wb := km.Kmer32ToBytes(c.Wrd[i])
		b.Write(wb)
		for j := 0; j < nc; j++ {
			b.WriteByte('\t')
			b.WriteString(strconv.FormatUint(uint64(c.Val[j][i]), 10))
		}
		b.WriteByte('\n')
	}
	b.Flush()
}

/*
	Kcounts32 methods
*/
func (cs *Kcounts32) SetName(s string, i int) {
	for j := 0; j < len(cs.Cou); j++ {
		if cs.Cou[j] != nil {
			cs.Cou[j].SetName(s, i)
		}
	}
}

// Throught counter routines
func (cs *Kcounts32) Count(seqChan chan []byte, couChan chan int) {
	for i := 0; i < len(cs.Cou); i++ {
		go cs.Cou[i].Count(seqChan, couChan)
	}
}

// Find all non nil counters and thought their ID in a channel
func (cs *Kcounts32) FindNonNil(paiChan chan int, max int) {
	n := 0
	i := 0
	for n < max && i < len(cs.Cou) {
		if cs.Cou[i] != nil {
			paiChan <- i
			n++
		}
		i++
	}
	if n < max {
		// Failed to find all expected counter
		// => this should not occure!
		panic("An issue occured while merging threaded counters.")
	}
}

// Merge a pair of counters
func (cs *Kcounts32) Merge(paiChan chan int, merChan chan int) {
	i := <-paiChan
	j := <-paiChan

	// Merging counters i and j...

	// Temp. variable
	totI := cs.Cou[i].Std
	totJ := cs.Cou[j].Std
	tmpN := 0
	tmpC := make([]uint32, totI+totJ)
	tmpI := make([]uint32, totI+totJ)

	idI := 0
	idJ := 0
	for idI < totI {
		// Save words from J if lower than current I
		for idJ < totJ && cs.Cou[i].Wrd[idI] > cs.Cou[j].Wrd[idJ] {
			tmpI[tmpN] = cs.Cou[j].Wrd[idJ]
			tmpC[tmpN] = cs.Cou[j].Val[0][idJ]
			idJ++
			tmpN++
		}
		// Cumul counter if they share the same word
		if idJ < totJ && cs.Cou[i].Wrd[idI] == cs.Cou[j].Wrd[idJ] {
			tmpI[tmpN] = cs.Cou[j].Wrd[idJ]
			tmpC[tmpN] = cs.Cou[j].Val[0][idJ] + cs.Cou[i].Val[0][idI]
			idI++
			idJ++
			tmpN++
		} else {
			// Only I has this word
			tmpI[tmpN] = cs.Cou[i].Wrd[idI]
			tmpC[tmpN] = cs.Cou[i].Val[0][idI]
			tmpN++
			idI++
		}
	}
	// Finish remaining words in J
	for idJ < totJ {
		// No need to check I
		tmpI[tmpN] = cs.Cou[j].Wrd[idJ]
		tmpC[tmpN] = cs.Cou[j].Val[0][idJ]
		idJ++
		tmpN++
	}

	// Erasing counter j
	cs.Cou[j] = nil

	// Reset counter i
	cs.Cou[i].Wrd = make([]uint32, tmpN)
	cs.Cou[i].Val[0] = make([]uint32, tmpN)
	cs.Cou[i].Std = tmpN
	for n := 0; n < tmpN; n++ {
		cs.Cou[i].Wrd[n] = tmpI[n]
		cs.Cou[i].Val[0][n] = tmpC[n]
	}

	tmpC = nil
	tmpI = nil

	merChan <- i
}

func (cs *Kcounts32) Write(output string) {
	for i := 0; i < len(cs.Cou); i++ {
		if cs.Cou[i] != nil {
			cs.Cou[i].Write(output)
			break
		}
	}
}

func (cs *Kcounts32) WriteAll(output string) {
	for i := 0; i < len(cs.Cou); i++ {
		if cs.Cou[i] != nil {
			cs.Cou[i].WriteAll(output)
			break
		}
	}
}
