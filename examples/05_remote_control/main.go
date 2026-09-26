// 05_remote_control: Remote control sequence — stop → latch clear → run.
//
// !! WARNING !!
// Running this program STOPS the connected PLC and then performs a latch clear,
// which ERASES the values of latched (power-failure retained) devices.
// The erased data cannot be recovered.
// Do NOT run while equipment is in operation or on a PLC holding data you need.
// If any step after the stop fails, the PLC is left in STOP.
// Type "yes" at the prompt to confirm before proceeding.
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
	fmt.Println("!! WARNING !! This program will STOP the PLC and ERASE latched device data.")
	fmt.Print("Continue? (yes/no): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.TrimSpace(scanner.Text()) != "yes" {
		fmt.Println("Aborted.")
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

	// RemoteStop puts the PLC CPU into STOP state.
	fmt.Println("stopping PLC...")
	if err := c.RemoteStop(); err != nil {
		panic(err)
	}
	fmt.Println("stopped")
	time.Sleep(500 * time.Millisecond)

	// RemoteLatchClear clears latched device values.
	// The PLC must already be stopped before calling this.
	fmt.Println("clearing latch...")
	if err := c.RemoteLatchClear(); err != nil {
		fmt.Fprintln(os.Stderr, "latch clear failed; the PLC remains in STOP:", err)
		os.Exit(1)
	}
	fmt.Println("latch cleared")
	time.Sleep(500 * time.Millisecond)

	// RemoteRun starts the PLC CPU.
	//   clearMode 0 — start without clearing memory (most common)
	//   clearMode 1 — clear non-latched devices before starting
	//   clearMode 2 — clear all devices before starting
	//   force false  — abort if another device is already in remote control
	fmt.Println("starting PLC...")
	if err := c.RemoteRun(0, false); err != nil {
		fmt.Fprintln(os.Stderr, "run failed; the PLC remains in STOP:", err)
		os.Exit(1)
	}
	fmt.Println("running")
}
