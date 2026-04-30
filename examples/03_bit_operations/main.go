// 03_bit_operations: ビットデバイス（M, X, Y）の読み書きサンプル。
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

	// X0 から 8 点読み取り（入力リレー）
	inputs, err := c.ReadBits("X", 0, 8)
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

	// M0 から 4 点書き込み（内部リレー）
	if err := c.WriteBits("M", 0, []bool{true, false, true, false}); err != nil {
		panic(err)
	}
	fmt.Println("M0-M3 書き込み完了")

	// 書き込んだ M0-M3 を読み返し
	bits, err := c.ReadBits("M", 0, 4)
	if err != nil {
		panic(err)
	}
	for i, b := range bits {
		fmt.Printf("M%d = %v\n", i, b)
	}
}
