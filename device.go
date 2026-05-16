package gomcprotocol

// Mode selects binary or ASCII framing.
type Mode int

const (
	ModeBinary Mode = iota // binary (3E/4E) framing
	ModeASCII              // ASCII (3E/4E) framing
)

// Binary device codes (iQ-R / Q series).
var binCode = map[string]byte{
	"D":  0xA8,
	"W":  0xB4,
	"R":  0xAF,
	"ZR": 0xB0,
	"X":  0x9C,
	"Y":  0x9D,
	"M":  0x90,
	"L":  0x92,
	"B":  0xA0,
	"F":  0x93,
	"TC": 0xC0,
	"TS": 0xC1,
	"CC": 0xC3,
	"CS": 0xC4,
	"SB": 0xA1,
	"SW": 0xB5,
	"SM": 0x91,
	"SD": 0xA9,
	"TN": 0xC2,
	"CN": 0xC5,
	"Z":  0xCC,
}

// wordDevs: word devices use decimal address in ASCII mode; bit devices use hex.
var wordDevs = map[string]bool{
	"D": true, "W": true, "R": true, "ZR": true,
	"TN": true, "CN": true, "Z": true, "SW": true, "SD": true,
}

var decimalAddrDevs = map[string]bool{
	"TC": true, "TS": true, "CC": true, "CS": true,
}

const (
	cmdRead  uint16 = 0x0401
	cmdWrite uint16 = 0x1401
	subcWord uint16 = 0x0000
	subcBit  uint16 = 0x0001
)
