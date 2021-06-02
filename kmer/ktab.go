package kmer

/*
	Basic functions to encode and decode data
	into binary ktab.
*/
// Convert a 16 bits uint into a slice of 2 bytes
func Uint16ToBytes(v uint16) []byte {
	out := make([]byte, 2)
	out[0] = byte(v >> 8)
	out[1] = byte(v & 0xFF)
	return out
}

// Convert a slice of bytes into a 16 bits uint
func BytesToUint16(v []byte) uint16 {
	out := uint16(v[0]) << 8
	out = out | uint16(v[1])
	return out
}

// Convert a 32 bits uint into a slice of 4 bytes
func Uint32ToBytes(v uint32) []byte {
	out := make([]byte, 4)
	for i := 0; i < 4; i++ {
		out[i] = byte((v >> ((3 - i) * 8)) & 0xFF)
	}
	return out
}

// Convert a slice of bytes into a 32 bits uint
func BytesToUint32(v []byte) uint32 {
	out := uint32(v[0])
	for i := 1; i < 4; i++ {
		out = (out << 8) | uint32(v[i])
	}
	return out
}

// Convert a 64 bits uint into a slice of 8 bytes
func Uint64ToBytes(v uint64) []byte {
	out := make([]byte, 8)
	for i := 0; i < 8; i++ {
		out[i] = byte((v >> ((7 - i) * 8)) & 0xFF)
	}
	return out
}

// Convert a slice of bytes into a 32 bits uint
func BytesToUint64(v []byte) uint64 {
	out := uint64(v[0])
	for i := 1; i < 8; i++ {
		out = (out << 8) | uint64(v[i])
	}
	return out
}

/*
	Ktab header management.
*/
type Khead struct {
	K      uint8
	Param  uint8
	Nlibs  uint16
	NWords uint64
	Names  []string
}

func NewKhead(k, p, l, w int) *Khead {
	return &Khead{
		uint8(k),
		uint8(p),
		uint16(l),
		uint64(w),
		make([]string, 0),
	}
}

// Add a name
func (kh *Khead) AddName(n string) {
	kh.Names = append(kh.Names, n)
}

// Set names
func (kh *Khead) SetNames(n []string) {
	kh.Names = n
}

// Encode a header into bytes
func (kh *Khead) Encode() []byte {
	out := make([]byte, 2)
	out[0] = kh.K
	out[1] = kh.Param
	out = append(out, Uint16ToBytes(kh.Nlibs)...)
	out = append(out, Uint64ToBytes(kh.NWords)...)
	return out
}

// Decode a header from bytes
func Decode(v []byte) *Khead {
	return &Khead{
		v[0],
		v[1],
		BytesToUint16(v[2:5]),
		BytesToUint64(v[6:11]),
		make([]string, 0),
	}
}

// Read/Load names