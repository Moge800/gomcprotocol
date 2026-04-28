package gomcprotocol

import "testing"

func TestRemoteRunBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	defer c.Close()
	if err := c.RemoteRun(0, false); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteRunAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, ""))
	defer done()
	c := connect(t, host, port, ModeASCII)
	defer c.Close()
	if err := c.RemoteRun(2, true); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteRunInvalidClearMode(t *testing.T) {
	c, _ := New3EClient("127.0.0.1", 1025, ModeBinary)
	if err := c.RemoteRun(3, false); err == nil {
		t.Fatal("expected error for invalid clearMode")
	}
}

func TestRemoteStopBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	defer c.Close()
	if err := c.RemoteStop(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteStopAsc(t *testing.T) {
	host, port, done := mockServer(t, ascResp(0, ""))
	defer done()
	c := connect(t, host, port, ModeASCII)
	defer c.Close()
	if err := c.RemoteStop(); err != nil {
		t.Fatal(err)
	}
}

func TestRemotePauseBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	defer c.Close()
	if err := c.RemotePause(false); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteLatchClearBin(t *testing.T) {
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	defer c.Close()
	if err := c.RemoteLatchClear(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteResetBin(t *testing.T) {
	// RemoteReset closes connection without waiting for response
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	// No error expected even if server closes connection
	c.RemoteReset()
}

func TestRemoteRunPLCError(t *testing.T) {
	host, port, done := mockServer(t, binResp(0xC059, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	defer c.Close()
	err := c.RemoteRun(0, false)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}
