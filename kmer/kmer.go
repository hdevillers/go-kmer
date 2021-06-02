package kmer

const (
	MaxKSmall    int = 10
	MaxK32Bits   int = 15
	MaxKPrintAll int = 12
	MaxK64Bits   int = 31
	MaxKAbsolute int = 31
)

type KmerCounter interface {
	Count(chan []byte, chan int)
	Merge(chan int, chan int)
	FindNonNil(chan int, int)
	SetName(string, int)
	Write(string)
	WriteAll(string)
}
