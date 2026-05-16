package gomcprotocol

// Mode selects binary or ASCII framing.
type Mode int

const (
	ModeBinary Mode = iota // binary (3E/4E) framing
	ModeASCII              // ASCII (3E/4E) framing
)

// Binary device codes (iQ-R / Q series).
var binCode = map[string]byte{
	"D":   0xA8,
	"W":   0xB4,
	"R":   0xAF,
	"ZR":  0xB0,
	"X":   0x9C,
	"Y":   0x9D,
	"M":   0x90,
	"L":   0x92,
	"B":   0xA0,
	"F":   0x93,
	"V":   0x94,
	"TC":  0xC0,
	"TS":  0xC1,
	"STC": 0xC6,
	"STS": 0xC7,
	"CC":  0xC3,
	"CS":  0xC4,
	"SB":  0xA1,
	"SW":  0xB5,
	"SM":  0x91,
	"SD":  0xA9,
	"TN":  0xC2,
	"STN": 0xC8,
	"CN":  0xC5,
	"S":   0x98,
	"DX":  0xA2,
	"DY":  0xA3,
	"Z":   0xCC,
}

// decimalAddrDevs records devices whose device numbers are decimal in ASCII mode.
var decimalAddrDevs = map[string]bool{
	"SM": true, "SD": true,
	"M": true, "L": true, "F": true, "V": true,
	"D": true, "R": true,
	"TC": true, "TS": true, "TN": true,
	"STC": true, "STS": true, "STN": true,
	"CC": true, "CS": true, "CN": true,
	"S": true, "Z": true,
}

var ascCode = map[string]string{
	"STC": "SC",
	"STS": "SS",
	"STN": "SN",
}

const (
	cmdRead  uint16 = 0x0401
	cmdWrite uint16 = 0x1401
	subcWord uint16 = 0x0000
	subcBit  uint16 = 0x0001
)
