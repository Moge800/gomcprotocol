// 04_random_access: Read and write multiple non-contiguous devices in one request.
//
// RandomRead and RandomWrite bundle multiple device addresses into a single
// MC Protocol packet, reducing round-trips compared to individual ReadWords calls.
// Use this when reading scattered addresses or mixing word (16-bit) and dword (32-bit) values.
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

	// RandomRead reads word (uint16) and dword (uint32) addresses in one request.
	// Pass nil for either slice if that type is not needed.
	words, dwords, err := c.RandomRead(
		[]mc.DeviceAddr{{"D", 100}, {"D", 200}}, // word reads:  D100, D200
		[]mc.DeviceAddr{{"D", 300}},              // dword read:  D300–D301 as uint32
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("D100         = %d\n", words[0])
	fmt.Printf("D200         = %d\n", words[1])
	fmt.Printf("D300 (dword) = %d\n", dwords[0])

	// RandomWrite writes word and dword values in one request.
	err = c.RandomWrite(
		[]mc.DeviceAddr{{"D", 100}, {"D", 200}}, // word addresses
		[]uint16{10, 20},                          // D100=10, D200=20
		[]mc.DeviceAddr{{"D", 300}},              // dword address
		[]uint32{100000},                          // D300–D301 = 100000
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("random write done")

	// RandomWriteBits writes to bit devices at scattered addresses in one request.
	err = c.RandomWriteBits(
		[]mc.DeviceAddr{{"M", 0}, {"M", 10}, {"Y", 5}}, // target addresses
		[]bool{true, false, true},                         // M0=ON, M10=OFF, Y5=ON
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("bit random write done")
}
