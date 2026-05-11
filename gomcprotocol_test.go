package gomcprotocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ── mock server ───────────────────────────────────────────────────────────────

// mockServer starts a TCP listener, serves one request with resp, then closes.
func mockServer(t *testing.T, resp []byte) (host string, port int, done func()) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 512)
		conn.Read(buf)
		conn.Write(resp)
	}()
	h, p, _ := net.SplitHostPort(l.Addr().String())
	port, _ = strconv.Atoi(p)
	return h, port, func() { l.Close() }
}

// binResp builds a binary-mode PLC response frame.
func binResp(endCode uint16, data []byte) []byte {
	payload := make([]byte, 2+len(data))
	binary.LittleEndian.PutUint16(payload, endCode)
	copy(payload[2:], data)
	resp := make([]byte, 9+len(payload))
	resp[0] = 0xD0
	resp[1] = 0x00
	binary.LittleEndian.PutUint16(resp[7:], uint16(len(payload)))
	copy(resp[9:], payload)
	return resp
}

// ascResp builds an ASCII-mode PLC response frame.
func ascResp(endCode uint16, data string) []byte {
	payload := fmt.Sprintf("%04X%s", endCode, data)
	s := fmt.Sprintf("D00000FF03FF00%04X%s", len(payload), payload)
	return []byte(s)
}

// connect creates a client connected to a mock server.
func connect(t *testing.T, host string, port int, mode Mode) *Client3E {
	t.Helper()
	c, err := New3EClient(host, port, mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	return c
}

func read3EBinRequest(conn net.Conn) error {
	header := make([]byte, 9)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	dataLen := int(binary.LittleEndian.Uint16(header[7:]))
	if dataLen > maxTestRequestDataLen {
		return fmt.Errorf("request payload too large: %d > %d", dataLen, maxTestRequestDataLen)
	}
	_, err := io.CopyN(io.Discard, conn, int64(dataLen))
	return err
}

func waitForStackContains(timeout time.Duration, needles ...string) error {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 1<<20)
	for time.Now().Before(deadline) {
		n := runtime.Stack(buf, true)
		stack := string(buf[:n])
		found := true
		for _, needle := range needles {
			if !strings.Contains(stack, needle) {
				found = false
				break
			}
		}
		if found {
			return nil
		}
		time.Sleep(time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for stack to contain %v", needles)
}

func readWords3EWithEnterSignal(c *Client3E, entered chan<- struct{}, device string, addr, count int) ([]uint16, error) {
	close(entered)
	return c.ReadWords(device, addr, count)
}

// ── frame building ────────────────────────────────────────────────────────────

func TestBuildBin(t *testing.T) {
	// ReadWords("D", 100, 10) in binary: timer=0x0010, cmd=0x0401, subcmd=0x0000
	// addr: 0x64 0x00 0x00 0xA8, count: 0x0A 0x00
	body := []byte{0x64, 0x00, 0x00, 0xA8, 0x0A, 0x00}
	got := buildBin(0x0010, 0x0401, 0x0000, body)

	want := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // header
		0x0C, 0x00, // data length = 12
		0x10, 0x00, // timer
		0x01, 0x04, // cmd
		0x00, 0x00, // subcmd
		0x64, 0x00, 0x00, 0xA8, 0x0A, 0x00, // body
	}
	if !bytes.Equal(got, want) {
		t.Errorf("buildBin mismatch\ngot:  %X\nwant: %X", got, want)
	}
}

func TestBuildAsc(t *testing.T) {
	// ReadWords("D", 100, 10): body = "D 000100000A"
	body := "D 000100000A"
	got := buildAsc(0x0010, 0x0401, 0x0000, body)
	// inner = "0010" + "0401" + "0000" + body = 12+12 = 24 chars
	// frame = "500000FF03FF00" + "0018" + inner
	want := "500000FF03FF000018" + "0010" + "0401" + "0000" + body
	if got != want {
		t.Errorf("buildAsc mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

// ── address encoding ──────────────────────────────────────────────────────────

func TestAddrBin(t *testing.T) {
	got := addrBin("D", 100)
	want := []byte{0x64, 0x00, 0x00, 0xA8}
	if !bytes.Equal(got, want) {
		t.Errorf("addrBin(D,100) = %X, want %X", got, want)
	}
}

func TestAddrBinLargeAddr(t *testing.T) {
	got := addrBin("ZR", 0x123456)
	want := []byte{0x56, 0x34, 0x12, 0xB0}
	if !bytes.Equal(got, want) {
		t.Errorf("addrBin(ZR,0x123456) = %X, want %X", got, want)
	}
}

func TestAddrAscWordDevice(t *testing.T) {
	// Word device: decimal address, device name padded to 2 chars
	if got, want := addrAsc("D", 100), "D 000100"; got != want {
		t.Errorf("addrAsc(D,100) = %q, want %q", got, want)
	}
}

func TestAddrAscBitDevice(t *testing.T) {
	// Bit device: hex address
	if got, want := addrAsc("M", 255), "M 0000FF"; got != want {
		t.Errorf("addrAsc(M,255) = %q, want %q", got, want)
	}
}

func TestAddrAscTwoCharDevice(t *testing.T) {
	if got, want := addrAsc("SB", 0), "SB000000"; got != want {
		t.Errorf("addrAsc(SB,0) = %q, want %q", got, want)
	}
}

// ── response checking ─────────────────────────────────────────────────────────

func TestChkBinOK(t *testing.T) {
	data := binResp(0, []byte{0x01, 0x00, 0x02, 0x00})
	got, err := chkBin(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte{0x01, 0x00, 0x02, 0x00}) {
		t.Errorf("chkBin data = %X", got)
	}
}

func TestChkBinError(t *testing.T) {
	data := binResp(0xC059, nil)
	_, err := chkBin(data)
	var mcErr *MCProtocolError
	if err == nil {
		t.Fatal("expected error")
	}
	if e, ok := err.(*MCProtocolError); !ok {
		t.Fatalf("expected *MCProtocolError, got %T", err)
	} else {
		mcErr = e
	}
	if mcErr.EndCode != 0xC059 {
		t.Errorf("EndCode = 0x%04X, want 0xC059", mcErr.EndCode)
	}
}

func TestChkBinShort(t *testing.T) {
	_, err := chkBin([]byte{0xD0, 0x00})
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

func TestChkAscOK(t *testing.T) {
	resp := string(ascResp(0, "00010002"))
	got, err := chkAsc(resp)
	if err != nil {
		t.Fatal(err)
	}
	if got != "00010002" {
		t.Errorf("chkAsc data = %q", got)
	}
}

func TestChkAscError(t *testing.T) {
	resp := string(ascResp(0xC059, ""))
	_, err := chkAsc(resp)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

func TestChkAscShort(t *testing.T) {
	_, err := chkAsc("D000")
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

// ── input validation ──────────────────────────────────────────────────────────

func TestValidateUnsupportedDevice(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	_, err := c.validate("Q", 0, 1)
	if err == nil {
		t.Fatal("expected error for unsupported device")
	}
}

func TestValidateNegativeStart(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	_, err := c.validate("D", -1, 1)
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}

func TestValidateZeroCount(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	_, err := c.validate("D", 0, 0)
	if err == nil {
		t.Fatal("expected error for zero count")
	}
}

func TestValidateCaseInsensitive(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	dev, err := c.validate("d", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dev != "D" {
		t.Errorf("expected D, got %q", dev)
	}
}

// ── error types ───────────────────────────────────────────────────────────────

func TestMCProtocolErrorMessage(t *testing.T) {
	err := &MCProtocolError{EndCode: 0xC059}
	if got, want := err.Error(), "MC error 0xC059"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestMCProtocolConnectionErrorMessage(t *testing.T) {
	err := connErr("not connected")
	if got := err.Error(); got != "not connected" {
		t.Errorf("Error() = %q", got)
	}
}

// ── ReadWords ─────────────────────────────────────────────────────────────────

func TestReadWordsBin(t *testing.T) {
	words := []uint16{100, 200, 300}
	data := make([]byte, len(words)*2)
	for i, w := range words {
		binary.LittleEndian.PutUint16(data[i*2:], w)
	}
	host, port, done := mockServer(t, binResp(0, data))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadWords("D", 100, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i, w := range words {
		if got[i] != w {
			t.Errorf("ReadWords[%d] = %d, want %d", i, got[i], w)
		}
	}
}

func TestReadWordsAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, "00640012"))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	got, err := c.ReadWords("D", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 0x0064 || got[1] != 0x0012 {
		t.Errorf("ReadWords = %v, want [0x64, 0x12]", got)
	}
}

func TestReadWordsBinPLCError(t *testing.T) {
	host, port, done := mockServer(t, binResp(0xC059, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	_, err := c.ReadWords("D", 0, 1)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

// ── WriteWords ────────────────────────────────────────────────────────────────

func TestWriteWordsBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	if err := c.WriteWords("D", 200, []uint16{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteWordsAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, ""))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	if err := c.WriteWords("D", 200, []uint16{0x1234}); err != nil {
		t.Fatal(err)
	}
}

// ── ReadBits ──────────────────────────────────────────────────────────────────

func TestReadBitsBin(t *testing.T) {
	// pymcprotocol encoding: even index → high nibble (bit4), odd index → low nibble (bit0)
	// byte 0: high nibble=1 (bit0=true),  low nibble=0 (bit1=false) → 0x10
	// byte 1: high nibble=1 (bit2=true),  low nibble=0 (bit3=false) → 0x10
	host, port, done := mockServer(t, binResp(0, []byte{0x10, 0x10}))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadBits("M", 0, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, true, false}
	for i, b := range want {
		if got[i] != b {
			t.Errorf("ReadBits[%d] = %v, want %v", i, got[i], b)
		}
	}
}

func TestReadBitsAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, "1010"))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	got, err := c.ReadBits("M", 0, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []bool{true, false, true, false}
	for i, b := range want {
		if got[i] != b {
			t.Errorf("ReadBits[%d] = %v, want %v", i, got[i], b)
		}
	}
}

func TestReadBitsOddCount(t *testing.T) {
	// 3 bits: byte 0 packs bits 0+1, byte 1 packs bit 2 only (high nibble)
	// byte 0: 0x11 → bit0=true (high), bit1=true (low)
	// byte 1: 0x10 → bit2=true (high)
	host, port, done := mockServer(t, binResp(0, []byte{0x11, 0x10}))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadBits("M", 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
}

// ── WriteBits ─────────────────────────────────────────────────────────────────

func TestWriteBitsBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	if err := c.WriteBits("Y", 0, []bool{true, false, true}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBitsAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, ""))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	if err := c.WriteBits("Y", 0, []bool{true, false, true}); err != nil {
		t.Fatal(err)
	}
}

// ── bit encoding round-trip ───────────────────────────────────────────────────

func TestBitPackingRoundTrip(t *testing.T) {
	// Verify write packing matches read unpacking for all 4 patterns.
	// pymcprotocol encoding: even → high nibble (bit4), odd → low nibble (bit0)
	cases := []struct {
		bits []bool
		want []byte
	}{
		{[]bool{false, false}, []byte{0x00}},
		{[]bool{true, false}, []byte{0x10}},
		{[]bool{false, true}, []byte{0x01}},
		{[]bool{true, true}, []byte{0x11}},
	}
	for _, tc := range cases {
		buf := make([]byte, 1)
		for i, v := range tc.bits {
			if v {
				if i%2 == 0 {
					buf[i/2] |= 0x10
				} else {
					buf[i/2] |= 0x01
				}
			}
		}
		if !bytes.Equal(buf, tc.want) {
			t.Errorf("pack(%v) = %X, want %X", tc.bits, buf, tc.want)
		}
		// unpack
		for i := range tc.bits {
			var got bool
			if i%2 == 0 {
				got = (buf[i/2]>>4)&0x01 != 0
			} else {
				got = buf[i/2]&0x01 != 0
			}
			if got != tc.bits[i] {
				t.Errorf("unpack bit[%d] = %v, want %v", i, got, tc.bits[i])
			}
		}
	}
}

// ── New3EClient validation ────────────────────────────────────────────────────

func TestNew3EClientInvalidMode(t *testing.T) {
	_, err := New3EClient("127.0.0.1", 1025, Mode(99))
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

// ── UDP transport ─────────────────────────────────────────────────────────────

// mockUDPServer starts a UDP listener, serves one datagram with resp, then closes.
func mockUDPServer(t *testing.T, resp []byte) (host string, port int, done func()) {
	t.Helper()
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		buf := make([]byte, 512)
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}
		conn.WriteTo(resp, addr)
	}()
	h, p, _ := net.SplitHostPort(conn.LocalAddr().String())
	port, _ = strconv.Atoi(p)
	return h, port, func() { conn.Close() }
}

func connectUDP(t *testing.T, host string, port int, mode Mode) *Client3E {
	t.Helper()
	c, err := New3EClientUDP(host, port, mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestReadWordsUDP(t *testing.T) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:], 100)
	binary.LittleEndian.PutUint16(data[2:], 200)
	host, port, done := mockUDPServer(t, binResp(0, data))
	defer done()

	c := connectUDP(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadWords("D", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 100 || got[1] != 200 {
		t.Errorf("ReadWords UDP = %v, want [100 200]", got)
	}
}

func TestWriteWordsUDP(t *testing.T) {
	host, port, done := mockUDPServer(t, binResp(0, nil))
	defer done()

	c := connectUDP(t, host, port, ModeBinary)
	defer c.Close()

	if err := c.WriteWords("D", 0, []uint16{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
}

func TestReadBitsUDP(t *testing.T) {
	host, port, done := mockUDPServer(t, binResp(0, []byte{0x10, 0x00}))
	defer done()

	c := connectUDP(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadBits("M", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0] || got[1] {
		t.Errorf("ReadBits UDP = %v, want [true false]", got)
	}
}

func TestWriteBitsUDP(t *testing.T) {
	host, port, done := mockUDPServer(t, binResp(0, nil))
	defer done()

	c := connectUDP(t, host, port, ModeBinary)
	defer c.Close()

	if err := c.WriteBits("M", 0, []bool{true, false}); err != nil {
		t.Fatal(err)
	}
}

func TestNew3EClientUDPInvalidMode(t *testing.T) {
	_, err := New3EClientUDP("127.0.0.1", 1025, Mode(99))
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestClient3ESerializesConcurrentRequests(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	firstRequest := make(chan struct{})
	secondReady := make(chan struct{})
	startSecond := make(chan struct{})
	secondCallEntered := make(chan struct{})
	serverErr := make(chan error, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()

		if err := read3EBinRequest(conn); err != nil {
			serverErr <- err
			return
		}
		close(firstRequest)

		select {
		case <-secondCallEntered:
		case <-time.After(time.Second):
			serverErr <- fmt.Errorf("timed out waiting for second request attempt")
			return
		}
		if err := waitForStackContains(time.Second, "readWords3EWithEnterSignal", "(*Client3E).sendBin"); err != nil {
			serverErr <- err
			return
		}
		if err := assertNoRequestBytes(conn, 75*time.Millisecond); err != nil {
			serverErr <- err
			return
		}

		if _, err := conn.Write(binResp(0, []byte{0x01, 0x00})); err != nil {
			serverErr <- err
			return
		}
		if err := read3EBinRequest(conn); err != nil {
			serverErr <- err
			return
		}
		_, err = conn.Write(binResp(0, []byte{0x02, 0x00}))
		serverErr <- err
	}()

	h, p, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(p)
	c := connect(t, h, port, ModeBinary)
	defer c.Close()

	firstDone := make(chan error, 1)
	go func() {
		got, err := c.ReadWords("D", 0, 1)
		if err != nil {
			firstDone <- err
			return
		}
		if len(got) != 1 || got[0] != 1 {
			firstDone <- fmt.Errorf("first ReadWords = %v, want [1]", got)
			return
		}
		firstDone <- nil
	}()

	select {
	case <-firstRequest:
	case err := <-serverErr:
		t.Fatalf("server failed before first request completed: %v", err)
	case err := <-firstDone:
		t.Fatalf("first request finished before server observed it: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first request")
	}

	secondDone := make(chan error, 1)
	go func() {
		close(secondReady)
		<-startSecond
		got, err := readWords3EWithEnterSignal(c, secondCallEntered, "D", 1, 1)
		if err != nil {
			secondDone <- err
			return
		}
		if len(got) != 1 || got[0] != 2 {
			secondDone <- fmt.Errorf("second ReadWords = %v, want [2]", got)
			return
		}
		secondDone <- nil
	}()

	select {
	case <-secondReady:
		close(startSecond)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second goroutine to become ready")
	}

	for _, step := range []struct {
		name string
		ch   <-chan error
	}{
		{"first read", firstDone},
		{"second read", secondDone},
		{"server", serverErr},
	} {
		select {
		case err := <-step.ch:
			if err != nil {
				t.Fatalf("%s failed: %v", step.name, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s", step.name)
		}
	}
}

func TestSetTimeout(t *testing.T) {
	// Start a server that reads the request but never replies and keeps the
	// connection open, so the client must time out via deadline (not EOF).
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 512)
		conn.Read(buf) // consume the request
		<-done         // hold connection open until test exits
	}()
	h, p, _ := net.SplitHostPort(l.Addr().String())
	port, _ := strconv.Atoi(p)
	c := connect(t, h, port, ModeBinary)
	defer c.Close()

	c.SetTimeout(50 * time.Millisecond)
	start := time.Now()
	_, err = c.ReadWords("D", 0, 1)
	elapsed := time.Since(start)
	if _, ok := err.(*MCProtocolConnectionError); !ok {
		t.Errorf("expected MCProtocolConnectionError on timeout, got %v", err)
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("expected timeout/deadline error, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~50ms)", elapsed)
	}
}
