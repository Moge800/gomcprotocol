package gomcprotocol

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"
)

// ── 4E mock server ────────────────────────────────────────────────────────────

// mock4EServer starts a TCP listener, serves one request with resp, then closes.
func mock4EServer(t *testing.T, resp []byte) (host string, port int, done func()) {
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

// bin4EResp builds a 4E binary-mode PLC response frame.
// Layout: D0 00 serial(2) 00 00 00 FF 03 FF 00 dataLen(2) endCode(2) data...
func bin4EResp(serial, endCode uint16, data []byte) []byte {
	payload := make([]byte, 2+len(data))
	binary.LittleEndian.PutUint16(payload, endCode)
	copy(payload[2:], data)
	resp := make([]byte, 13+len(payload))
	resp[0] = 0xD4
	resp[1] = 0x00
	binary.LittleEndian.PutUint16(resp[2:], serial)
	// resp[4:6] reserved = 0x00
	resp[6] = 0x00
	resp[7] = 0xFF
	resp[8] = 0xFF
	resp[9] = 0x03
	resp[10] = 0x00
	binary.LittleEndian.PutUint16(resp[11:], uint16(len(payload)))
	copy(resp[13:], payload)
	return resp
}

// asc4EResp builds a 4E ASCII-mode PLC response frame.
func asc4EResp(serial, endCode uint16, data string) []byte {
	payload := fmt.Sprintf("%04X%s", endCode, data)
	s := fmt.Sprintf("D400%04X000000FF03FF00%04X%s", serial, len(payload), payload)
	return []byte(s)
}

func connect4E(t *testing.T, host string, port int, mode Mode) *Client4E {
	t.Helper()
	c, err := New4EClient(host, port, mode)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	return c
}

// ── frame building ────────────────────────────────────────────────────────────

func TestBuild4EBin(t *testing.T) {
	c := &Client4E{timer: 0x0010}
	// cmd=0x0401 subcmd=0x0000 body=[0x64 0x00 0x00 0xA8 0x0A 0x00]
	payload := []byte{0x01, 0x04, 0x00, 0x00, 0x64, 0x00, 0x00, 0xA8, 0x0A, 0x00}
	frame := c.build4EBin(0x0001, payload)
	if frame[0] != 0x54 || frame[1] != 0x00 {
		t.Errorf("subheader = %02X%02X, want 5400", frame[0], frame[1])
	}
	serial := binary.LittleEndian.Uint16(frame[2:])
	if serial != 0x0001 {
		t.Errorf("serial = %04X, want 0001", serial)
	}
	dataLen := binary.LittleEndian.Uint16(frame[11:])
	// inner = timer(2) + payload
	if int(dataLen) != 2+len(payload) {
		t.Errorf("dataLen = %d, want %d", dataLen, 2+len(payload))
	}
}

func TestBuild4EAsc(t *testing.T) {
	c := &Client4E{timer: 0x0010}
	body := "040100000010040000000A"
	got := c.build4EAsc(0x0001, body)
	if got[:4] != "5400" {
		t.Errorf("subheader = %q, want 5400", got[:4])
	}
	if got[4:8] != "0001" {
		t.Errorf("serial = %q, want 0001", got[4:8])
	}
}

// ── New4EClient validation ────────────────────────────────────────────────────

func TestNew4EClientInvalidMode(t *testing.T) {
	_, err := New4EClient("127.0.0.1", 1025, Mode(99))
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

// ── chk4EBin / chk4EAsc ──────────────────────────────────────────────────────

func TestChk4EBinOK(t *testing.T) {
	resp := bin4EResp(1, 0, []byte{0x01, 0x00})
	got, err := chk4EBin(resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 0x01 {
		t.Errorf("chk4EBin data = %X", got)
	}
}

func TestChk4EBinError(t *testing.T) {
	resp := bin4EResp(1, 0xC059, nil)
	_, err := chk4EBin(resp)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

func TestChk4EBinShort(t *testing.T) {
	_, err := chk4EBin([]byte{0xD4, 0x00})
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

func TestChk4EAscOK(t *testing.T) {
	resp := string(asc4EResp(1, 0, "ABCD"))
	got, err := chk4EAsc(resp)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ABCD" {
		t.Errorf("chk4EAsc data = %q", got)
	}
}

func TestChk4EAscError(t *testing.T) {
	resp := string(asc4EResp(1, 0xC059, ""))
	_, err := chk4EAsc(resp)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

func TestChk4EAscShort(t *testing.T) {
	_, err := chk4EAsc("D400")
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

// ── ReadWords ─────────────────────────────────────────────────────────────────

func TestReadWords4EBin(t *testing.T) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:], 100)
	binary.LittleEndian.PutUint16(data[2:], 200)
	host, port, done := mock4EServer(t, bin4EResp(1, 0, data))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadWords("D", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 100 || got[1] != 200 {
		t.Errorf("ReadWords = %v, want [100 200]", got)
	}
}

func TestReadWords4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, "00640012"))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	got, err := c.ReadWords("D", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != 0x0064 || got[1] != 0x0012 {
		t.Errorf("ReadWords = %v, want [0x64, 0x12]", got)
	}
}

func TestReadWords4EBinPLCError(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0xC059, nil))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	_, err := c.ReadWords("D", 0, 1)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

// ── WriteWords ────────────────────────────────────────────────────────────────

func TestWriteWords4EBin(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0, nil))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	if err := c.WriteWords("D", 0, []uint16{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteWords4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, ""))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	if err := c.WriteWords("D", 0, []uint16{0x1234}); err != nil {
		t.Fatal(err)
	}
}

// ── ReadBits ──────────────────────────────────────────────────────────────────

func TestReadBits4EBin(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0, []byte{0x10, 0x00}))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	got, err := c.ReadBits("M", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0] || got[1] {
		t.Errorf("ReadBits = %v, want [true false]", got)
	}
}

func TestReadBits4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, "10"))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	got, err := c.ReadBits("M", 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !got[0] || got[1] {
		t.Errorf("ReadBits = %v, want [true false]", got)
	}
}

// ── WriteBits ─────────────────────────────────────────────────────────────────

func TestWriteBits4EBin(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0, nil))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	if err := c.WriteBits("Y", 0, []bool{true, false, true}); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBits4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, ""))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	if err := c.WriteBits("Y", 0, []bool{true, false}); err != nil {
		t.Fatal(err)
	}
}

// ── RandomRead ────────────────────────────────────────────────────────────────

func TestRandomRead4EBin(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:], 100)
	binary.LittleEndian.PutUint16(data[2:], 200)
	binary.LittleEndian.PutUint32(data[4:], 100000)
	host, port, done := mock4EServer(t, bin4EResp(1, 0, data))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	wVals, dVals, err := c.RandomRead(
		[]DeviceAddr{{"D", 100}, {"D", 200}},
		[]DeviceAddr{{"D", 300}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if wVals[0] != 100 || wVals[1] != 200 {
		t.Errorf("word values = %v, want [100 200]", wVals)
	}
	if dVals[0] != 100000 {
		t.Errorf("dword value = %d, want 100000", dVals[0])
	}
}

func TestRandomRead4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, "ABCD"))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	wVals, dVals, err := c.RandomRead([]DeviceAddr{{"D", 0}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if wVals[0] != 0xABCD {
		t.Errorf("word value = 0x%04X, want 0xABCD", wVals[0])
	}
	if len(dVals) != 0 {
		t.Errorf("dVals should be empty")
	}
}

// ── RandomWrite ───────────────────────────────────────────────────────────────

func TestRandomWrite4EBin(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0, nil))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	err := c.RandomWrite(
		[]DeviceAddr{{"D", 100}},
		[]uint16{42},
		[]DeviceAddr{{"D", 200}},
		[]uint32{100000},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWrite4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, ""))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	err := c.RandomWrite(
		[]DeviceAddr{{"D", 0}},
		[]uint16{0x1234},
		nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWrite4ELengthMismatch(t *testing.T) {
	c, _ := New4EClient("127.0.0.1", 1025, ModeBinary)
	err := c.RandomWrite(
		[]DeviceAddr{{"D", 0}},
		[]uint16{1, 2},
		nil, nil,
	)
	if err == nil {
		t.Fatal("expected error for length mismatch")
	}
}

// ── RandomWriteBits ───────────────────────────────────────────────────────────

func TestRandomWriteBits4EBin(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0, nil))
	defer done()

	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()

	err := c.RandomWriteBits(
		[]DeviceAddr{{"M", 0}, {"M", 10}},
		[]bool{true, false},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWriteBits4EAsc(t *testing.T) {
	host, port, done := mock4EServer(t, asc4EResp(1, 0, ""))
	defer done()

	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()

	err := c.RandomWriteBits(
		[]DeviceAddr{{"M", 0}},
		[]bool{true},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWriteBits4ELengthMismatch(t *testing.T) {
	c, _ := New4EClient("127.0.0.1", 1025, ModeBinary)
	err := c.RandomWriteBits(
		[]DeviceAddr{{"M", 0}, {"M", 1}},
		[]bool{true},
	)
	if err == nil {
		t.Fatal("expected error for length mismatch")
	}
}

// ── input validation ──────────────────────────────────────────────────────────

func TestValidate4EUnsupportedDevice(t *testing.T) {
	c, _ := New4EClient("127.0.0.1", 1025, ModeBinary)
	_, err := c.validate("Q", 0, 1)
	if err == nil {
		t.Fatal("expected error for unsupported device")
	}
}

func TestValidate4ENegativeStart(t *testing.T) {
	c, _ := New4EClient("127.0.0.1", 1025, ModeBinary)
	_, err := c.validate("D", -1, 1)
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}

func TestValidate4ECaseInsensitive(t *testing.T) {
	c, _ := New4EClient("127.0.0.1", 1025, ModeBinary)
	dev, err := c.validate("d", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dev != "D" {
		t.Errorf("expected D, got %q", dev)
	}
}

// ── serial number ─────────────────────────────────────────────────────────────

func TestSerialNumberIncrement(t *testing.T) {
	c := &Client4E{}
	s1 := c.nextSerial()
	s2 := c.nextSerial()
	if s2 != s1+1 {
		t.Errorf("serial not incrementing: %d, %d", s1, s2)
	}
}

func TestSerialNumberWrapAround(t *testing.T) {
	c := &Client4E{serialNo: 0xFFFF}
	s := c.nextSerial()
	if s != 1 {
		t.Errorf("expected wrap to 1, got %d", s)
	}
}

func Test4ESetTimeout(t *testing.T) {
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
	c := connect4E(t, h, port, ModeBinary)
	defer c.Close()

	c.SetTimeout(50 * time.Millisecond)
	start := time.Now()
	_, err = c.ReadWords("D", 0, 1)
	elapsed := time.Since(start)
	if _, ok := err.(*MCProtocolConnectionError); !ok {
		t.Errorf("expected MCProtocolConnectionError on timeout, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v (expected ~50ms)", elapsed)
	}
}
