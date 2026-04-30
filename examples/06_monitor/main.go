// 06_monitor: 複数デバイスを定期ポーリングし、値が変化したら表示するサンプル。
// 接続が切れた場合は自動で再接続を試みる。
// Ctrl+C で終了。
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	mc "github.com/moge800/gomcprotocol"
)

const (
	host        = "192.168.0.1"
	port        = 5007
	interval    = 500 * time.Millisecond
	retryDelay  = 3 * time.Second
)

// monitored はポーリング対象のワードデバイス一覧。
var monitored = []mc.DeviceAddr{
	{"D", 100},
	{"D", 101},
	{"D", 200},
}

func connect() (*mc.Client3E, error) {
	c, err := mc.New3EClient(host, port, mc.ModeBinary)
	if err != nil {
		return nil, err
	}
	c.SetTimeout(3 * time.Second)
	if err := c.Connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// waitOrQuit は d の間待機する。quit シグナルを受けたら false を返す。
func waitOrQuit(quit <-chan os.Signal, d time.Duration) bool {
	select {
	case <-quit:
		return false
	case <-time.After(d):
		return true
	}
}

func main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	prev := make([]uint16, len(monitored))
	var c *mc.Client3E

	fmt.Println("モニタ開始（Ctrl+C で終了）")

	for {
		select {
		case <-quit:
			fmt.Println("\n終了")
			if c != nil {
				c.Close()
			}
			return
		case <-ticker.C:
		}

		// 未接続なら接続を試みる
		if c == nil {
			var err error
			c, err = connect()
			if err != nil {
				fmt.Fprintf(os.Stderr, "接続失敗: %v — %v 後に再試行\n", err, retryDelay)
				if !waitOrQuit(quit, retryDelay) {
					fmt.Println("\n終了")
					return
				}
				continue
			}
			fmt.Println("接続しました")
		}

		// ランダムリードで全デバイスを一括取得
		words, _, err := c.RandomRead(monitored, nil)
		if err != nil {
			var connErr *mc.MCProtocolConnectionError
			var plcErr *mc.MCProtocolError
			switch {
			case errors.As(err, &connErr):
				fmt.Fprintf(os.Stderr, "通信エラー: %v — 再接続します\n", err)
				c.Close()
				c = nil
			case errors.As(err, &plcErr):
				fmt.Fprintf(os.Stderr, "PLC エラー (end code 0x%04X): %v\n", plcErr.EndCode, err)
			default:
				fmt.Fprintf(os.Stderr, "クライアントエラー: %v\n", err)
			}
			continue
		}

		// 変化があったデバイスのみ表示
		for i, addr := range monitored {
			if words[i] != prev[i] {
				fmt.Printf("[%s] %s%d: %d → %d\n",
					time.Now().Format("15:04:05"),
					addr.Device, addr.Addr,
					prev[i], words[i],
				)
				prev[i] = words[i]
			}
		}
	}
}
