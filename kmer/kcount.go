package kmer

import (
	"bufio"
	"math"
	"os"
	"strconv"
)

type Kcount struct {
	K   int        // Value of K
	Id  int        // counter ID
	Con []uint32   // Byte to base convertion
	Fwd int        // forward operator
	Bwd int        // Backward operator
	Val [][]uint32 // counted values
	Nam []string   // Name of the libs
	Ust bool       // Unstranded kmer count (boolean)
}

func NewKcount(id int, k int, ust bool) *Kcount {
	var c Kcount
	c.K = k
	c.Id = id
	c.Con = make([]uint32, 256)
	c.Fwd = (16 - k + 1) * 2
	c.Bwd = (16 - k) * 2
	c.Val = make([][]uint32, 1)
	c.Val[0] = make([]uint32, int(math.Pow(4.0, float64(k))))
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

type Kcounts struct {
	Cou []*Kcount // An array of counters
}

func NewKcounts(th int, k int, ust bool) *Kcounts {
	// th is the number of counting threads
	var cs Kcounts
	cs.Cou = make([]*Kcount, th)
	for i := 0; i < th; i++ {
		cs.Cou[i] = NewKcount(i, k, ust)
	}
	return &cs
}

/*
	Kcount methods
*/
// Set the name of a counter
func (c *Kcount) SetName(s string, i int) {
	if i >= len(c.Nam) {
		panic("Name index to large, you must extend the counter first.")
	}
	c.Nam[i] = s
}

// Extend counter
func (c *Kcount) Extend(n int) {
	i := len(c.Val)

	// Extend the value container
	c.Val = append(c.Val, make([][]uint32, n)...)
	for i < len(c.Val) {
		c.Val[i] = make([]uint32, int(math.Pow(4.0, float64(c.K))))
		i++
	}

	// Extend the name container
	c.Nam = append(c.Nam, make([]string, n)...)
}

// Count words from sequences of bytes provided by a channel
func (c *Kcount) Count(seqChan chan []byte, couChan chan int) {
	// Read channel
	for seq := range seqChan {
		// Length of the input sequence
		l := len(seq)

		// The input sequence must contain at least K bases
		if l >= c.K {
			// Init the first word
			w := uint32(0)
			for i := 0; i < c.K; i++ {
				w = (w << 2) | c.Con[seq[i]]
			}

			// Count the first word
			c.Val[0][w]++

			// Continue with the following words
			for i := c.K; i < l; i++ {
				w = (w<<c.Fwd)>>c.Bwd | c.Con[seq[i]]
				c.Val[0][w]++
			}
		}
	}

	// Merge identical words if unstranded
	if c.Ust {
		nw := uint32(len(c.Val[0]))
		skip := make([]bool, nw)
		i := uint32(0)
		km := NewKmer32(c.K)
		for i < nw {
			if skip[i] {
				i++
				continue
			} else {
				irc := km.Kmer32RevComp(i)
				skip[i] = true
				skip[irc] = true
				if i < irc {
					c.Val[0][i] = c.Val[0][i] + c.Val[0][irc]
					c.Val[0][irc] = 0
				}
				i++
			}
		}
	}

	// No more sequences in the channel
	couChan <- c.Id
}

// Write the Kmer counter into a file
func (c *Kcount) Write(output string) {
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
	for i := 0; i < len(c.Val[0]); i++ {
		sum := uint32(0)
		for j := 0; j < nc; j++ {
			sum += c.Val[j][i]
		}
		if sum > 0 {
			// Convert uint32 words into bytes
			wb := km.Kmer32ToBytes(uint32(i))
			b.Write(wb)
			for j := 0; j < nc; j++ {
				b.WriteByte('\t')
				b.WriteString(strconv.FormatUint(uint64(c.Val[j][i]), 10))
			}
			b.WriteByte('\n')
		}
	}
	b.Flush()
}

// Write all counter values including null values
func (c *Kcount) WriteAll(output string) {
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
	for i := 0; i < len(c.Val[0]); i++ {
		// Convert uint32 words into bytes
		wb := km.Kmer32ToBytes(uint32(i))
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
	Kcounts methods
*/
// Throught counter routines
func (cs *Kcounts) Count(seqChan chan []byte, couChan chan int) {
	for i := 0; i < len(cs.Cou); i++ {
		go cs.Cou[i].Count(seqChan, couChan)
	}
}

// Find all non-nil counters (and throught there ID in a channel)
func (cs *Kcounts) FindNonNil(paiChan chan int, max int) {
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

// Merge pair of counters
func (cs *Kcounts) Merge(paiChan chan int, merChan chan int) {
	i := <-paiChan
	j := <-paiChan

	// Cumulate counter values
	l := len(cs.Cou[i].Val[0])
	for n := 0; n < l; n++ {
		cs.Cou[i].Val[0][n] += cs.Cou[j].Val[0][n]
	}

	// Delete counter j
	cs.Cou[j] = nil

	// Throught end of merging
	merChan <- i
}

func (cs *Kcounts) SetName(s string, i int) {
	for j := 0; j < len(cs.Cou); j++ {
		if cs.Cou[j] != nil {
			cs.Cou[j].SetName(s, i)
		}
	}
}

func (cs *Kcounts) Write(output string) {
	for i := 0; i < len(cs.Cou); i++ {
		if cs.Cou[i] != nil {
			cs.Cou[i].Write(output)
			break
		}
	}
}

func (cs *Kcounts) WriteAll(output string) {
	for i := 0; i < len(cs.Cou); i++ {
		if cs.Cou[i] != nil {
			cs.Cou[i].WriteAll(output)
			break
		}
	}
}
