package gomcprotocol

import (
	"fmt"
	"net"
	"testing"
)

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
	host, port, done := mockServer(t, binResp(0, nil))
	defer done()
	c := connect(t, host, port, ModeBinary)
	if err := c.RemoteReset(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteResetBinEarlyClose(t *testing.T) {
	// Simulate PLC closing the connection immediately after receiving the command,
	// without sending a response — the expected behavior of a real PLC reset.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		buf := make([]byte, 512)
		conn.Read(buf)
		conn.Close()
	}()
	host, portStr, _ := net.SplitHostPort(l.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	c := connect(t, host, port, ModeBinary)
	// A connection error is expected: PLC closes before responding.
	err = c.RemoteReset()
	if _, ok := err.(*MCProtocolConnectionError); !ok {
		t.Errorf("expected MCProtocolConnectionError, got %v", err)
	}
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
