package gomcprotocol

import (
	"encoding/binary"
	"testing"
)

// mock1E builds a 1E response frame: [end_code 2B LE][data].
func mock1E(endCode uint16, data []byte) []byte {
	resp := make([]byte, 2+len(data))
	binary.LittleEndian.PutUint16(resp, endCode)
	copy(resp[2:], data)
	return resp
}

// connect1E creates a Client1E connected to a mock server.
func connect1E(t *testing.T, host string, port int) *Client1E {
	t.Helper()
	c := New1EClient(host, port)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	return c
}

// ── frame building ────────────────────────────────────────────────────────────

func TestBuild1EFrame(t *testing.T) {
	c := New1EClient("127.0.0.1", 1025)
	body := body1E("D", 100, 10)
	frame := c.build1E(sub1EWordRead, body)

	// [subheader=0x01][PC=0xFF][timer_L=0x10][timer_H=0x00][addr 3B][devcode 1B][count 2B]
	if frame[0] != 0x01 {
		t.Errorf("subheader = 0x%02X, want 0x01", frame[0])
	}
	if frame[1] != 0xFF {
		t.Errorf("PC No = 0x%02X, want 0xFF", frame[1])
	}
	if frame[2] != 0x10 || frame[3] != 0x00 {
		t.Errorf("timer = [0x%02X 0x%02X], want [0x10 0x00]", frame[2], frame[3])
	}
	// addr D100: 0x64 0x00 0x00, devcode 0xA8, count 0x0A 0x00
	wantBody := []byte{0x64, 0x00, 0x00, 0xA8, 0x0A, 0x00}
	for i, b := range wantBody {
		if frame[4+i] != b {
			t.Errorf("body[%d] = 0x%02X, want 0x%02X", i, frame[4+i], b)
		}
	}
}

func TestChk1EOK(t *testing.T) {
	data := mock1E(0, []byte{0x01, 0x00, 0x02, 0x00})
	got, err := chk1E(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Errorf("len(data) = %d, want 4", len(got))
	}
}

func TestChk1EError(t *testing.T) {
	data := mock1E(0xC059, nil)
	_, err := chk1E(data)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

func TestChk1EShort(t *testing.T) {
	_, err := chk1E([]byte{0x00})
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

// ── ReadWords ─────────────────────────────────────────────────────────────────

func TestClient1EReadWords(t *testing.T) {
	words := []uint16{100, 200, 300}
	data := make([]byte, len(words)*2)
	for i, w := range words {
		binary.LittleEndian.PutUint16(data[i*2:], w)
	}
	host, port, done := mockServer(t, mock1E(0, data))
	defer done()

	c := connect1E(t, host, port)
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

func TestClient1EReadWordsPLCError(t *testing.T) {
	host, port, done := mockServer(t, mock1E(0xC059, nil))
	defer done()

	c := connect1E(t, host, port)
	defer c.Close()

	_, err := c.ReadWords("D", 0, 1)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

// ── WriteWords ────────────────────────────────────────────────────────────────

func TestClient1EWriteWords(t *testing.T) {
	host, port, done := mockServer(t, mock1E(0, nil))
	defer done()

	c := connect1E(t, host, port)
	defer c.Close()

	if err := c.WriteWords("D", 100, []uint16{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
}

// ── ReadBits ──────────────────────────────────────────────────────────────────

func TestClient1EReadBits(t *testing.T) {
	// 1E bit format: 1 byte per point (0x00=OFF, non-zero=ON)
	host, port, done := mockServer(t, mock1E(0, []byte{0x01, 0x00, 0x01, 0x00}))
	defer done()

	c := connect1E(t, host, port)
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

// ── WriteBits ─────────────────────────────────────────────────────────────────

func TestClient1EWriteBits(t *testing.T) {
	host, port, done := mockServer(t, mock1E(0, nil))
	defer done()

	c := connect1E(t, host, port)
	defer c.Close()

	if err := c.WriteBits("Y", 0, []bool{true, false, true}); err != nil {
		t.Fatal(err)
	}
}

// ── validation ────────────────────────────────────────────────────────────────

func TestClient1EValidateUnsupportedDevice(t *testing.T) {
	c := New1EClient("127.0.0.1", 1025)
	_, err := c.validate("Q", 0, 1)
	if err == nil {
		t.Fatal("expected error for unsupported device")
	}
}

func TestClient1EValidateNegativeStart(t *testing.T) {
	c := New1EClient("127.0.0.1", 1025)
	_, err := c.validate("D", -1, 1)
	if err == nil {
		t.Fatal("expected error for negative start")
	}
}
