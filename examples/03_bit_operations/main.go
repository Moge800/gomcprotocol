// 03_bit_operations: Read and write bit devices (X, M, Y).
//
// Bit devices represent single on/off signals:
//
//	X — input relay  (reflects physical input terminals; typically read-only)
//	Y — output relay (drives physical output terminals)
//	M — internal relay (general-purpose; freely readable and writable)
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

	// ReadBits reads `count` consecutive bit values starting at `start`.
	// X is the input relay — reflects the state of physical input terminals.
	inputs, err := c.ReadBits("X", 0, 8) // X0–X7
	if err != nil {
		panic(err)
	}
	fmt.Print("X0-X7: ")
	for _, b := range inputs {
		if b {
			fmt.Print("1 ")
		} else {
			fmt.Print("0 ")
		}
	}
	fmt.Println()

	// WriteBits writes a slice of bool values to consecutive bit addresses.
	// M is an internal relay commonly used for internal PLC logic.
	// Use an address that is known to be unused when testing on a real PLC.
	if err := c.WriteBits("M", 0, []bool{true, false, true, false}); err != nil { // M0–M3
		panic(err)
	}
	fmt.Println("wrote M0-M3")

	// Read back M0–M3 to confirm the values.
	bits, err := c.ReadBits("M", 0, 4)
	if err != nil {
		panic(err)
	}
	for i, b := range bits {
		fmt.Printf("M%d = %v\n", i, b)
	}
}
