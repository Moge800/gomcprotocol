// 01_basic_read: Minimal example reading word devices from a PLC.
//
// Connects to a PLC at 192.168.0.1:5007 using binary-mode 3E frame,
// reads 5 consecutive word registers starting at D100, and prints them.
//
// Expected output:
//
//	D100 = 0
//	D101 = 0
//	D102 = 0
//	D103 = 0
//	D104 = 0
package main

import (
	"fmt"

	mc "github.com/moge800/gomcprotocol"
)

func main() {
	// New3EClient creates a 3E-frame client. ModeBinary is compact and recommended.
	// Call Connect() before any read/write operations.
	c, err := mc.New3EClient("192.168.0.1", 5007, mc.ModeBinary)
	if err != nil {
		panic(err)
	}
	if err := c.Connect(); err != nil {
		panic(err)
	}
	defer c.Close()

	// ReadWords reads `count` consecutive 16-bit word values starting at `start`.
	// D is the data register — the most commonly used word device.
	words, err := c.ReadWords("D", 100, 5) // D100–D104
	if err != nil {
		panic(err)
	}
	for i, v := range words {
		fmt.Printf("D%d = %d\n", 100+i, v)
	}
}
