// 01_basic_read: PLCからワードデバイスを読み取る最小限のサンプル。
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

	// D100 から 5 点読み取り
	words, err := c.ReadWords("D", 100, 5)
	if err != nil {
		panic(err)
	}
	for i, v := range words {
		fmt.Printf("D%d = %d\n", 100+i, v)
	}
}
