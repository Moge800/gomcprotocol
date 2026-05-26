# gomcprotocol

[English version](./README.md)

三菱電機 PLC と MC プロトコル（SLMP）で通信するための Go ライブラリです。

## 機能

- **3E フレーム** — TCP / UDP トランスポート、バイナリ / ASCII モード対応
- **4E フレーム** — TCP、バイナリ / ASCII モード対応、シリアル番号管理付き
- バッチ読み書き：ワード・ビット
- ランダム読み書き：複数デバイスを1リクエストで処理
- リモートコントロール：RUN / STOP / PAUSE / ラッチクリア / リセット
- goroutine セーフ：同一クライアントインスタンス上のリクエストを内部でシリアライズ

## REST Wrapper API

このライブラリを REST API として利用できる wrapper が [Moge800/gomc-rest](https://github.com/Moge800/gomc-rest) にあります。
他のアプリケーションやサービスから HTTP 経由で MC プロトコル通信を扱いたい場合に利用できます。

## インストール

```bash
go get github.com/moge800/gomcprotocol@latest

# メジャーリリースを明示して固定する場合
go get github.com/moge800/gomcprotocol@v1
```

```go
import "github.com/moge800/gomcprotocol"
```

## クイックスタート

### 3E フレーム（TCP）

```go
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

    // D100 から 5 ワード読み出し
    words, err := c.ReadWords("D", 100, 5)
    if err != nil {
        panic(err)
    }
    fmt.Println(words)

    // D200 に書き込み
    if err := c.WriteWords("D", 200, []uint16{1, 2, 3}); err != nil {
        panic(err)
    }
}
```

### 3E フレーム（UDP）

```go
c, err := mc.New3EClientUDP("192.168.0.1", 5007, mc.ModeBinary)
```

### 4E フレーム

```go
c, err := mc.New4EClient("192.168.0.1", 5007, mc.ModeBinary)
```

## API リファレンス

### Client3E

| メソッド | 説明 |
|---------|------|
| `New3EClient(host, port, mode)` | 3E フレーム TCP クライアント作成 |
| `New3EClientUDP(host, port, mode)` | 3E フレーム UDP クライアント作成 |
| `Connect()` | 接続確立 |
| `Close()` | 接続クローズ |
| `ReadWords(device, start, count)` | ワード一括読み出し |
| `WriteWords(device, start, values)` | ワード一括書き込み |
| `ReadBits(device, start, count)` | ビット一括読み出し |
| `WriteBits(device, start, values)` | ビット一括書き込み |
| `RandomRead(words, dwords)` | 複数デバイス一括読み出し |
| `RandomWrite(words, wordVals, dwords, dwordVals)` | 複数デバイス一括書き込み |
| `RandomWriteBits(devices, values)` | 複数ビットデバイス一括書き込み |
| `RemoteRun(clearMode, force)` | PLC リモート RUN |
| `RemoteStop()` | PLC リモート STOP |
| `RemotePause(force)` | PLC リモート PAUSE |
| `RemoteLatchClear()` | ラッチクリア（STOP 状態で実行） |
| `RemoteReset()` | PLC リモートリセット（接続クローズ） |

`Client4E` も `Client3E` と同じメソッドを提供します。`New4EClient` で作成してください。

同一クライアントインスタンスに対して同時に呼び出されたリクエストは内部で直列化されるため、その接続上で同時に実行される MC プロトコルのリクエスト／レスポンス交換は常に 1 件です。

### モード

| 定数 | 説明 |
|------|------|
| `ModeBinary` | バイナリモード（コンパクト、推奨） |
| `ModeASCII` | ASCII モード（人が読みやすい） |

### 対応デバイス

| デバイス | 種別 | 説明 |
|---------|------|------|
| `D` | ワード | データレジスタ |
| `W` | ワード | リンクレジスタ |
| `R` | ワード | ファイルレジスタ |
| `ZR` | ワード | ファイルレジスタ（拡張） |
| `SW` | ワード | リンク特殊レジスタ |
| `TN` | ワード | タイマ現在値 |
| `STN` | ワード | 積算タイマ現在値 |
| `CN` | ワード | カウンタ現在値 |
| `Z` | ワード | インデックスレジスタ |
| `X` | ビット | 入力 |
| `Y` | ビット | 出力 |
| `M` | ビット | 内部リレー |
| `L` | ビット | ラッチリレー |
| `V` | ビット | エッジリレー |
| `S` | ビット | ステップリレー |
| `DX` | ビット | ダイレクトアクセス入力 |
| `DY` | ビット | ダイレクトアクセス出力 |
| `TC` | ビット | タイマコイル |
| `TS` | ビット | タイマ接点 |
| `STC` | ビット | 積算タイマコイル |
| `STS` | ビット | 積算タイマ接点 |
| `CC` | ビット | カウンタコイル |
| `CS` | ビット | カウンタ接点 |
| `B` | ビット | リンクリレー |
| `F` | ビット | アナンシェータ |
| `SB` | ビット | リンク特殊リレー |
| `SM` | ビット | 特殊リレー |
| `SD` | ワード | 特殊レジスタ |

デバイス名は大文字小文字を区別しません（`"d"` と `"D"` はどちらも有効）。

### ランダムアクセス

```go
// D100（ワード）、D200（ワード）、D300（ダブルワード）を1リクエストで読み出し
words, dwords, err := c.RandomRead(
    []mc.DeviceAddr{{"D", 100}, {"D", 200}},
    []mc.DeviceAddr{{"D", 300}},
)

// D100=10、D200=20（ワード）、D300=100000（ダブルワード）を書き込み
err = c.RandomWrite(
    []mc.DeviceAddr{{"D", 100}, {"D", 200}},
    []uint16{10, 20},
    []mc.DeviceAddr{{"D", 300}},
    []uint32{100000},
)

// M0、M10、Y5 にビット書き込み
err = c.RandomWriteBits(
    []mc.DeviceAddr{{"M", 0}, {"M", 10}, {"Y", 5}},
    []bool{true, false, true},
)
```

### エラーハンドリング

```go
words, err := c.ReadWords("D", 100, 5)
if err != nil {
    if mcErr, ok := err.(*mc.MCProtocolError); ok {
        // PLC がエラーコードを返した場合
        fmt.Printf("PLC エラー: 0x%04X\n", mcErr.EndCode)
    } else {
        // ネットワーク／接続エラー
        fmt.Println("接続エラー:", err)
    }
}
```

## PLC 側の設定

PLC 側で Ethernet 通信および SLMP（MC プロトコル）を有効にしてください。  
Q シリーズ / iQ-R シリーズのデフォルトポートは通常 `5007` です。

## サンプルコード

実行可能なサンプルが [`examples/`](./examples/) ディレクトリにあります：

| ディレクトリ | 内容 |
|-------------|------|
| [`01_basic_read`](./examples/01_basic_read/main.go) | ワードデバイスの読み取り（最小構成） |
| [`02_basic_write`](./examples/02_basic_write/main.go) | ワードの書き込みと読み返し |
| [`03_bit_operations`](./examples/03_bit_operations/main.go) | ビットデバイス（X, M, Y）の読み書き |
| [`04_random_access`](./examples/04_random_access/main.go) | 複数デバイスの一括読み書き |
| [`05_remote_control`](./examples/05_remote_control/main.go) | 停止→ラッチクリア→起動のシーケンス |
| [`06_monitor`](./examples/06_monitor/main.go) | 定期ポーリング＋変化検出＋自動再接続 |

## ライセンス

Apache License, Version 2.0
