package main

import (
	"flag"
	"log"
	"os"

	"github.com/hdevillers/go-kmer/kmer"
	"github.com/hdevillers/go-seq/seqio"
)

/* Define an input type to allow multiple input files */
type inputFlags []string

func (i *inputFlags) String() string {
	return "hello, world\n"
}

func (i *inputFlags) Set(s string) error {
	*i = append(*i, s)
	return nil
}

var input inputFlags

func main() {
	k := flag.Int("k", 4, "K value.")
	flag.Var(&input, "i", "Input sequence file(s).")
	f := flag.String("f", "fasta", "Input sequence format.")
	o := flag.String("o", "kmer.tab", "Output file name.")
	d := flag.Bool("d", false, "Decompress the input (gz).")
	u := flag.Bool("unstranded", false, "Count Kmer in unstranded mode.")
	n := flag.String("name", "lib", "Name of the library.")
	a := flag.Bool("all", false, "Print all Kmers, including zero-count.")
	threads := flag.Int("threads", 4, "Number of threads.")
	flag.Parse()

	logger := log.New(os.Stderr, "DEBUG: ", log.Lmicroseconds)

	if len(input) == 0 {
		panic("You must provide at one input fasta file.")
	}

	if *a {
		if *k > kmer.MaxKPrintAll {
			panic("K value is too large to print all possible Kmers.")
		}
	}

	// Number of requiered channel
	seqChan := make(chan []byte, *threads)
	couChan := make(chan int)
	paiChan := make(chan int)
	merChan := make(chan int)

	// Determine the type of counter
	var kmerCounter kmer.KmerCounter
	logger.Print("Initializing counter...")
	if *k <= kmer.MaxKSmall {
		kmerCounter = kmer.NewKcounts(*threads, *k, *u)
	} else if *k <= kmer.MaxK32Bits {
		kmerCounter = kmer.NewKcounts32(*threads, *k, *u)
	} else {
		panic("K value is too large.")
	}

	logger.Print("Start reading sequences...")
	// Launch couting routines
	kmerCounter.Count(seqChan, couChan)

	// For each input files
	for i := 0; i < len(input); i++ {
		// Read input sequences
		seqIn := seqio.NewReader(input[i], *f, *d)
		// Count Kmer in all input sequences
		for seqIn.Next() {
			seqIn.CheckPanic()
			s := seqIn.Seq()
			seqChan <- s.Sequence
		}
	}
	close(seqChan)

	// Wait until all counters are done
	for j := 0; j < *threads; j++ {
		<-couChan
	}

	// Merge multi-threaded counters
	nc := *threads // Number of counters
	nm := nc / 2   // Number of merging process
	rm := nc % 2   // Number of unmerged counter
	for nc > 1 {
		// merging go routine
		for i := 0; i < nm; i++ {
			go kmerCounter.Merge(paiChan, merChan)
		}

		// through pairs
		kmerCounter.FindNonNil(paiChan, 2*nm)

		// Wait the merged counters
		for i := 0; i < nm; i++ {
			<-merChan
		}

		// refine numbers
		nc = nm + rm
		nm = nc / 2
		rm = nc % 2
	}

	// Set the lib name in the counter
	kmerCounter.SetName(*n, 0)

	logger.Print("Start writing out...")
	// Print out counted value
	if *a {
		kmerCounter.WriteAll(*o)
	} else {
		kmerCounter.Write(*o)
	}
	logger.Print("Finished.")
}
