// 02_basic_write: ワードデバイスへの書き込みと読み返しのサンプル。
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

	// D200 から 3 点書き込み
	writeVals := []uint16{100, 200, 300}
	if err := c.WriteWords("D", 200, writeVals); err != nil {
		panic(err)
	}
	fmt.Println("書き込み完了:", writeVals)

	// 書き込んだ値を読み返して確認
	readVals, err := c.ReadWords("D", 200, 3)
	if err != nil {
		panic(err)
	}
	fmt.Println("読み返し結果:", readVals)
}
