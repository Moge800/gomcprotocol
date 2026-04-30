// 04_random_access: バラバラなアドレスをまとめて1リクエストで読み書きするサンプル。
// 複数のデバイスを個別にアクセスするより通信回数を減らせる。
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

	// D100, D200 (ワード) と D300 (ダブルワード) を1リクエストで読み取り
	words, dwords, err := c.RandomRead(
		[]mc.DeviceAddr{{"D", 100}, {"D", 200}},
		[]mc.DeviceAddr{{"D", 300}},
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("D100 = %d\n", words[0])
	fmt.Printf("D200 = %d\n", words[1])
	fmt.Printf("D300 (dword) = %d\n", dwords[0])

	// D100=10, D200=20 (ワード) と D300=100000 (ダブルワード) を1リクエストで書き込み
	err = c.RandomWrite(
		[]mc.DeviceAddr{{"D", 100}, {"D", 200}},
		[]uint16{10, 20},
		[]mc.DeviceAddr{{"D", 300}},
		[]uint32{100000},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("ランダム書き込み完了")

	// M0, M10, Y5 のビットを1リクエストで書き込み
	err = c.RandomWriteBits(
		[]mc.DeviceAddr{{"M", 0}, {"M", 10}, {"Y", 5}},
		[]bool{true, false, true},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("ビットランダム書き込み完了")
}
