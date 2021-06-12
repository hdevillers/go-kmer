package kmer

type Ktab interface {
}

/*
	Parameter flags (encoded into a uint8)
	* 0x01 Stranded/Unstranded (true means unstranded);
	* 0x02 Print all option (true means print all);
	* 0x04 Unused yet;
	* 0x08 Unused yet;
	* 0x10 Unused yet;
	* 0x20 Unused yet;
	* 0x40 Unused yet;
	* 0x80 Unused yet;
*/
func SetParameter(ust bool, pal bool) uint8 {
	p := uint8(0)

	if ust {
		p = p | 0x01
	}

	if pal {
		p = p | 0x02
	}

	return p
}

func IsUnstranded(p uint8) bool {
	return p&0x01 == 0x01
}

func DoPrintAll(p uint8) bool {
	return p&0x02 == 0x02
}
