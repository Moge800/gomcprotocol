// 06_monitor: Poll multiple devices on a fixed interval and print value changes.
//
//   - Polls D100, D101, D200 every 500 ms using RandomRead (one request per cycle).
//   - Prints a timestamped line only when a value changes since the last cycle.
//   - Reconnects automatically on connection loss; retries every 3 s.
//   - Terminates cleanly on Ctrl+C (SIGINT/SIGTERM).
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
	host       = "192.168.0.1"
	port       = 5007
	interval   = 500 * time.Millisecond
	retryDelay = 3 * time.Second
)

// monitored is the list of word devices to watch.
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
	c.SetTimeout(3 * time.Second) // per-request deadline
	if err := c.Connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// waitOrQuit waits for duration d or a quit signal, whichever comes first.
// Returns false if the quit signal was received.
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

	// Ticker drives the poll cycle. The select below handles both tick and quit
	// without blocking — no sleep, no goroutine per device.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	prev := make([]uint16, len(monitored)) // last-seen values
	var c *mc.Client3E                     // nil = not connected

	fmt.Println("monitoring (Ctrl+C to quit)")

	for {
		// Block until the next tick or a quit signal.
		select {
		case <-quit:
			fmt.Println("\nquitting")
			if c != nil {
				c.Close()
			}
			return
		case <-ticker.C:
		}

		// Reconnect if the connection was lost.
		if c == nil {
			var err error
			c, err = connect()
			if err != nil {
				fmt.Fprintf(os.Stderr, "connect failed: %v — retrying in %v\n", err, retryDelay)
				// waitOrQuit instead of time.Sleep so Ctrl+C stays responsive during the wait.
				if !waitOrQuit(quit, retryDelay) {
					fmt.Println("\nquitting")
					return
				}
				continue
			}
			fmt.Println("connected")
		}

		// RandomRead fetches all monitored devices in a single MC Protocol request.
		words, _, err := c.RandomRead(monitored, nil)
		if err != nil {
			var connErr *mc.MCProtocolConnectionError
			var plcErr *mc.MCProtocolError
			switch {
			case errors.As(err, &connErr):
				// Network-level failure — drop the connection and reconnect next cycle.
				fmt.Fprintf(os.Stderr, "connection error: %v — reconnecting\n", err)
				c.Close()
				c = nil
			case errors.As(err, &plcErr):
				// PLC returned a non-zero end code (e.g. invalid device address).
				fmt.Fprintf(os.Stderr, "PLC error (end code 0x%04X): %v\n", plcErr.EndCode, err)
			default:
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
			}
			continue
		}

		// Print only addresses whose value changed since the last cycle.
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
