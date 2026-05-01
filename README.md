# gomcprotocol

[日本語版はこちら](./README_jp.md)

A Go library for communicating with Mitsubishi PLCs using the MC Protocol (SLMP).

## Features

- **3E frame** — TCP and UDP transport, Binary and ASCII mode
- **4E frame** — TCP, Binary and ASCII mode, with serial number tracking
- Batch read/write: words and bits
- Random read/write: multiple devices in a single request
- Remote control: Run, Stop, Pause, Latch Clear, Reset
- Goroutine-safe: requests are serialized internally

## REST Wrapper API

A REST API wrapper for this library is available at [Moge800/gomc-rest](https://github.com/Moge800/gomc-rest).  
Use it when you want to expose MC Protocol communication over HTTP from another application or service.

## Installation

```bash
go get github.com/moge800/gomcprotocol
```

```go
import "github.com/moge800/gomcprotocol"
```

## Quick Start

### 3E frame (TCP)

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

    // Read 5 words from D100
    words, err := c.ReadWords("D", 100, 5)
    if err != nil {
        panic(err)
    }
    fmt.Println(words)

    // Write to D200
    if err := c.WriteWords("D", 200, []uint16{1, 2, 3}); err != nil {
        panic(err)
    }
}
```

### 3E frame (UDP)

```go
c, err := mc.New3EClientUDP("192.168.0.1", 5007, mc.ModeBinary)
```

### 4E frame

```go
c, err := mc.New4EClient("192.168.0.1", 5007, mc.ModeBinary)
```

## API Reference

### Client3E

| Method | Description |
|--------|-------------|
| `New3EClient(host, port, mode)` | Create a 3E TCP client |
| `New3EClientUDP(host, port, mode)` | Create a 3E UDP client |
| `Connect()` | Establish connection |
| `Close()` | Close connection |
| `ReadWords(device, start, count)` | Read word values |
| `WriteWords(device, start, values)` | Write word values |
| `ReadBits(device, start, count)` | Read bit values |
| `WriteBits(device, start, values)` | Write bit values |
| `RandomRead(words, dwords)` | Read multiple devices at once |
| `RandomWrite(words, wordVals, dwords, dwordVals)` | Write multiple devices at once |
| `RandomWriteBits(devices, values)` | Write multiple bit devices at once |
| `RemoteRun(clearMode, force)` | Start PLC remotely |
| `RemoteStop()` | Stop PLC remotely |
| `RemotePause(force)` | Pause PLC remotely |
| `RemoteLatchClear()` | Clear latch (PLC must be stopped) |
| `RemoteReset()` | Reset PLC (connection will close) |

`Client4E` provides the same methods as `Client3E` (except remote commands), created via `New4EClient`.

### Modes

| Constant | Description |
|----------|-------------|
| `ModeBinary` | Binary framing (compact, recommended) |
| `ModeASCII` | ASCII framing (human-readable) |

### Supported Devices

| Device | Type | Notes |
|--------|------|-------|
| `D` | Word | Data register |
| `W` | Word | Link register |
| `R` | Word | File register |
| `ZR` | Word | File register (extended) |
| `SW` | Word | Link special register |
| `TN` | Word | Timer current value |
| `CN` | Word | Counter current value |
| `Z` | Word | Index register |
| `X` | Bit | Input |
| `Y` | Bit | Output |
| `M` | Bit | Internal relay |
| `L` | Bit | Latch relay |
| `B` | Bit | Link relay |
| `F` | Bit | Annunciator |
| `SB` | Bit | Link special relay |
| `SM` | Bit | Special relay |
| `SD` | Word | Special register |

Device names are case-insensitive (`"d"` and `"D"` both work).

### Random Access

```go
// Read D100 (word) and D200 (word) and D300 (dword) in one request
words, dwords, err := c.RandomRead(
    []mc.DeviceAddr{{"D", 100}, {"D", 200}},
    []mc.DeviceAddr{{"D", 300}},
)

// Write D100=10, D200=20 (words) and D300=100000 (dword)
err = c.RandomWrite(
    []mc.DeviceAddr{{"D", 100}, {"D", 200}},
    []uint16{10, 20},
    []mc.DeviceAddr{{"D", 300}},
    []uint32{100000},
)

// Write bits to M0, M10, Y5
err = c.RandomWriteBits(
    []mc.DeviceAddr{{"M", 0}, {"M", 10}, {"Y", 5}},
    []bool{true, false, true},
)
```

### Error Handling

```go
words, err := c.ReadWords("D", 100, 5)
if err != nil {
    if mcErr, ok := err.(*mc.MCProtocolError); ok {
        // PLC returned a non-zero end code
        fmt.Printf("PLC error: 0x%04X\n", mcErr.EndCode)
    } else {
        // Network or connection error
        fmt.Println("connection error:", err)
    }
}
```

## PLC Setup

The PLC must have Ethernet communication enabled with SLMP (MC Protocol) configured.  
Default port is typically `5007` for Q/iQ-R series.

## Examples

Runnable examples are in the [`examples/`](./examples/) directory:

| Directory | Description |
|-----------|-------------|
| [`01_basic_read`](./examples/01_basic_read/main.go) | Read word devices (minimal) |
| [`02_basic_write`](./examples/02_basic_write/main.go) | Write words and read back |
| [`03_bit_operations`](./examples/03_bit_operations/main.go) | Read/write bit devices (X, M, Y) |
| [`04_random_access`](./examples/04_random_access/main.go) | RandomRead/RandomWrite across multiple devices |
| [`05_remote_control`](./examples/05_remote_control/main.go) | Stop → LatchClear → Run sequence |
| [`06_monitor`](./examples/06_monitor/main.go) | Polling loop with change detection and auto-reconnect |

## License

MIT
