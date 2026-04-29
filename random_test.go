package gomcprotocol

import (
	"encoding/binary"
	"testing"
)

// ── RandomRead ────────────────────────────────────────────────────────────────

func TestRandomReadBin(t *testing.T) {
	// 2 words (D100, D200) + 1 dword (D300) → 4+4 = 8 data bytes
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:], 0x0064) // D100 = 100
	binary.LittleEndian.PutUint16(data[2:], 0x00C8) // D200 = 200
	binary.LittleEndian.PutUint32(data[4:], 0x000186A0) // D300 = 100000

	host, port, done := mockServer(t, binResp(0, data))
	defer done()

	c := connect(t, host, port, ModeBinary)
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

func TestRandomReadAsc(t *testing.T) {
	// 1 word (D0=0xABCD) + 0 dwords
	host, port, done := mockServer(t, ascResp(0, "ABCD"))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	wVals, dVals, err := c.RandomRead(
		[]DeviceAddr{{"D", 0}},
		nil,
	)
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

func TestRandomReadBinPLCError(t *testing.T) {
	host, port, done := mockServer(t, binResp(0xC059, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	_, _, err := c.RandomRead([]DeviceAddr{{"D", 0}}, nil)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}

func TestRandomReadEmptySlices(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	wVals, dVals, err := c.RandomRead(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(wVals) != 0 || len(dVals) != 0 {
		t.Errorf("expected empty results")
	}
}

// ── RandomWrite ───────────────────────────────────────────────────────────────

func TestRandomWriteBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	err := c.RandomWrite(
		[]DeviceAddr{{"D", 100}, {"D", 200}},
		[]uint16{10, 20},
		[]DeviceAddr{{"D", 300}},
		[]uint32{100000},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWriteAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, ""))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	err := c.RandomWrite(
		[]DeviceAddr{{"D", 0}},
		[]uint16{0x1234},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWriteLengthMismatch(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	err := c.RandomWrite(
		[]DeviceAddr{{"D", 0}},
		[]uint16{1, 2}, // mismatched length
		nil, nil,
	)
	if err == nil {
		t.Fatal("expected error for length mismatch")
	}
}

// ── RandomWriteBits ───────────────────────────────────────────────────────────

func TestRandomWriteBitsBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()

	c := connect(t, host, port, ModeBinary)
	defer c.Close()

	err := c.RandomWriteBits(
		[]DeviceAddr{{"M", 0}, {"M", 10}, {"Y", 5}},
		[]bool{true, false, true},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWriteBitsAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, ""))
	defer done()

	c := connect(t, host, port, ModeASCII)
	defer c.Close()

	err := c.RandomWriteBits(
		[]DeviceAddr{{"M", 0}},
		[]bool{true},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRandomWriteBitsLengthMismatch(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	err := c.RandomWriteBits(
		[]DeviceAddr{{"M", 0}, {"M", 1}},
		[]bool{true}, // mismatched length
	)
	if err == nil {
		t.Fatal("expected error for length mismatch")
	}
}

func TestRandomWriteBitsInvalidDevice(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	err := c.RandomWriteBits(
		[]DeviceAddr{{"Q", 0}}, // unsupported device
		[]bool{true},
	)
	if err == nil {
		t.Fatal("expected error for invalid device")
	}
}
