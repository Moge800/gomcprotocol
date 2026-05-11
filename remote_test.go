package gomcprotocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
)

const max4ETestRequestDataLen = 1024

func mock4ERequestServer(t *testing.T, mode Mode, resp []byte, check func([]byte) error) (host string, port int, done func()) {
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
		var request []byte
		if mode == ModeBinary {
			request, err = read4EBinRequestFrame(conn)
		} else {
			request, err = read4EAscRequestFrame(conn)
		}
		if err != nil {
			t.Errorf("read 4E request: %v", err)
			return
		}
		if err := check(request); err != nil {
			t.Error(err)
		}
		conn.Write(resp)
	}()
	host, portStr, _ := net.SplitHostPort(l.Addr().String())
	port, _ = strconv.Atoi(portStr)
	return host, port, func() { l.Close() }
}

func read4EBinRequestFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 13)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	dataLen := int(binary.LittleEndian.Uint16(header[11:]))
	if dataLen > max4ETestRequestDataLen {
		return nil, fmt.Errorf("request payload too large: %d > %d", dataLen, max4ETestRequestDataLen)
	}
	body := make([]byte, dataLen)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, err
	}
	return append(header, body...), nil
}

func read4EAscRequestFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 26)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	dataLen, err := strconv.ParseUint(string(header[22:26]), 16, 16)
	if err != nil {
		return nil, err
	}
	if dataLen > max4ETestRequestDataLen {
		return nil, fmt.Errorf("request payload too large: %d > %d", dataLen, max4ETestRequestDataLen)
	}
	body := make([]byte, int(dataLen))
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, err
	}
	return append(header, body...), nil
}

func assert4EBinRemoteRequest(request []byte, command uint16, body []byte) error {
	if len(request) < 19 {
		return fmt.Errorf("short 4E request: %X", request)
	}
	payload := request[13:]
	if got := binary.LittleEndian.Uint16(payload[2:]); got != command {
		return fmt.Errorf("command = 0x%04X, want 0x%04X", got, command)
	}
	if got := binary.LittleEndian.Uint16(payload[4:]); got != 0x0000 {
		return fmt.Errorf("subcommand = 0x%04X, want 0x0000", got)
	}
	if got := payload[6:]; !bytes.Equal(got, body) {
		return fmt.Errorf("body = %X, want %X", got, body)
	}
	return nil
}

func assert4EAscRemoteRequest(request []byte, command string, body string) error {
	if len(request) < 38 {
		return fmt.Errorf("short 4E ASCII request: %q", request)
	}
	want := "0010" + command + "0000" + body
	if got := string(request[26:]); got != want {
		return fmt.Errorf("payload = %q, want %q", got, want)
	}
	return nil
}

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

func TestRemoteRun4EBinRequest(t *testing.T) {
	host, port, done := mock4ERequestServer(t, ModeBinary, bin4EResp(1, 0, nil), func(request []byte) error {
		return assert4EBinRemoteRequest(request, 0x1001, []byte{0x03, 0x00, 0x02, 0x00})
	})
	defer done()
	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()
	if err := c.RemoteRun(2, true); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteRun4EAscRequest(t *testing.T) {
	host, port, done := mock4ERequestServer(t, ModeASCII, asc4EResp(1, 0, ""), func(request []byte) error {
		return assert4EAscRemoteRequest(request, "1001", "00030200")
	})
	defer done()
	c := connect4E(t, host, port, ModeASCII)
	defer c.Close()
	if err := c.RemoteRun(2, true); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteRun4EInvalidClearMode(t *testing.T) {
	c, _ := New4EClient("127.0.0.1", 1025, ModeBinary)
	if err := c.RemoteRun(3, false); err == nil {
		t.Fatal("expected error for invalid clearMode")
	}
}

func TestRemoteOperations4EBinRequests(t *testing.T) {
	tests := []struct {
		name    string
		command uint16
		body    []byte
		call    func(*Client4E) error
	}{
		{name: "stop", command: 0x1002, body: []byte{0x01, 0x00}, call: func(c *Client4E) error { return c.RemoteStop() }},
		{name: "pause", command: 0x1003, body: []byte{0x03, 0x00}, call: func(c *Client4E) error { return c.RemotePause(true) }},
		{name: "latch clear", command: 0x1005, body: []byte{0x01, 0x00}, call: func(c *Client4E) error { return c.RemoteLatchClear() }},
		{name: "reset", command: 0x1006, body: []byte{0x01, 0x00}, call: func(c *Client4E) error { return c.RemoteReset() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, done := mock4ERequestServer(t, ModeBinary, bin4EResp(1, 0, nil), func(request []byte) error {
				return assert4EBinRemoteRequest(request, tt.command, tt.body)
			})
			defer done()
			c := connect4E(t, host, port, ModeBinary)
			defer c.Close()
			if err := tt.call(c); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRemoteOperations4EAscRequests(t *testing.T) {
	tests := []struct {
		name    string
		command string
		body    string
		call    func(*Client4E) error
	}{
		{name: "stop", command: "1002", body: "0001", call: func(c *Client4E) error { return c.RemoteStop() }},
		{name: "pause", command: "1003", body: "0003", call: func(c *Client4E) error { return c.RemotePause(true) }},
		{name: "latch clear", command: "1005", body: "0001", call: func(c *Client4E) error { return c.RemoteLatchClear() }},
		{name: "reset", command: "1006", body: "0001", call: func(c *Client4E) error { return c.RemoteReset() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, done := mock4ERequestServer(t, ModeASCII, asc4EResp(1, 0, ""), func(request []byte) error {
				return assert4EAscRemoteRequest(request, tt.command, tt.body)
			})
			defer done()
			c := connect4E(t, host, port, ModeASCII)
			defer c.Close()
			if err := tt.call(c); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRemoteReset4EBinEarlyClose(t *testing.T) {
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
		_, _ = read4EBinRequestFrame(conn)
		conn.Close()
	}()
	host, portStr, _ := net.SplitHostPort(l.Addr().String())
	port := 0
	fmt.Sscanf(portStr, "%d", &port)
	c := connect4E(t, host, port, ModeBinary)
	err = c.RemoteReset()
	if _, ok := err.(*MCProtocolConnectionError); !ok {
		t.Errorf("expected MCProtocolConnectionError, got %v", err)
	}
}

func TestRemoteRun4EPLCError(t *testing.T) {
	host, port, done := mock4EServer(t, bin4EResp(1, 0xC059, nil))
	defer done()
	c := connect4E(t, host, port, ModeBinary)
	defer c.Close()
	err := c.RemoteRun(0, false)
	if e, ok := err.(*MCProtocolError); !ok || e.EndCode != 0xC059 {
		t.Errorf("expected MCProtocolError(0xC059), got %v", err)
	}
}
