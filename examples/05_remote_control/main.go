// 05_remote_control: PLCのリモート操作（停止・ラッチクリア・起動）サンプル。
//
// !! 警告 !!
// このプログラムを実行すると接続先の PLC が停止します。
// 設備・装置が動作中の場合は絶対に実行しないでください。
// 実行前に安全を確認し、"yes" と入力して続行してください。
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	mc "github.com/moge800/gomcprotocol"
)

func main() {
	fmt.Println("!! 警告 !! このプログラムは PLC を停止させます。")
	fmt.Print("続行しますか？ (yes/no): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.TrimSpace(scanner.Text()) != "yes" {
		fmt.Println("中止しました。")
		return
	}

	c, err := mc.New3EClient("192.168.0.1", 5007, mc.ModeBinary)
	if err != nil {
		panic(err)
	}
	if err := c.Connect(); err != nil {
		panic(err)
	}
	defer c.Close()

	// PLC を停止
	fmt.Println("PLC を停止中...")
	if err := c.RemoteStop(); err != nil {
		panic(err)
	}
	fmt.Println("停止完了")
	time.Sleep(500 * time.Millisecond)

	// ラッチクリア（停止中のみ可能）
	fmt.Println("ラッチクリア中...")
	if err := c.RemoteLatchClear(); err != nil {
		panic(err)
	}
	fmt.Println("ラッチクリア完了")
	time.Sleep(500 * time.Millisecond)

	// PLC を起動（clearMode=0: クリアなし）
	fmt.Println("PLC を起動中...")
	if err := c.RemoteRun(0, false); err != nil {
		panic(err)
	}
	fmt.Println("起動完了")
}
