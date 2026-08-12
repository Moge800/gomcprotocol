// 02_basic_write: Write word values to the PLC and read them back to verify.
//
// Writes three values to D200–D202, then reads them back.
// Reading back after writing is a simple way to confirm the operation succeeded.
package main

import (
	"fmt"

	mc "github.com/moge800/gomcprotocol"
)

func main() {
	c, err := mc.New3EClient("192.168.0.1", 5007, mc.ModeBinary)
	if err != nil {
		panic(err)
	}
	if err := c.Connect(); err != nil {
		panic(err)
	}
	defer c.Close()

	// WriteWords writes a slice of uint16 values to consecutive addresses.
	writeVals := []uint16{100, 200, 300}
	if err := c.WriteWords("D", 200, writeVals); err != nil { // D200–D202
		panic(err)
	}
	fmt.Println("write:", writeVals)

	// Read back to confirm the values were written correctly.
	readVals, err := c.ReadWords("D", 200, 3)
	if err != nil {
		panic(err)
	}
	fmt.Println("read back:", readVals)
}
