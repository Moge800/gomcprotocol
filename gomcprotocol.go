// Package gomcprotocol implements Mitsubishi MC Protocol clients for 3E and 4E frames
// over TCP and UDP transports.
package gomcprotocol

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client3E is a 3E frame MC Protocol client (TCP or UDP).
// Safe for concurrent use; requests are serialized by an internal mutex.
type Client3E struct {
	mu      sync.Mutex
	host    string
	port    int
	mode    Mode
	timeout time.Duration
	timer   uint16
	isUDP   bool
	conn    net.Conn
}

// New3EClient creates a new 3E frame client. Call Connect before use.
func New3EClient(host string, port int, mode Mode) (*Client3E, error) {
	if mode != ModeBinary && mode != ModeASCII {
		return nil, fmt.Errorf("mode must be ModeBinary or ModeASCII")
	}
	return &Client3E{
		host:    host,
		port:    port,
		mode:    mode,
		timeout: 5 * time.Second,
		timer:   0x0010,
	}, nil
}

// New3EClientUDP creates a new 3E frame client using UDP transport.
// Call Connect before use.
func New3EClientUDP(host string, port int, mode Mode) (*Client3E, error) {
	c, err := New3EClient(host, port, mode)
	if err != nil {
		return nil, err
	}
	c.isUDP = true
	return c, nil
}

// Connect establishes the connection to the PLC (TCP or UDP).
func (c *Client3E) Connect() error {
	proto := "tcp"
	if c.isUDP {
		proto = "udp"
	}
	conn, err := net.DialTimeout(proto, fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return &MCProtocolConnectionError{msg: "connect: " + err.Error()}
	}
	c.conn = conn
	return nil
}

// sendBin sends a binary frame and returns the response.
// Acquires the mutex to serialize concurrent callers.
func (c *Client3E) sendBin(frame []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, connErr("not connected")
	}
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	if c.isUDP {
		return xferBinUDP(c.conn, frame)
	}
	return xferBin(c.conn, frame)
}

// sendAsc sends an ASCII frame and returns the response.
// Acquires the mutex to serialize concurrent callers.
func (c *Client3E) sendAsc(frame string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return "", connErr("not connected")
	}
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	if c.isUDP {
		return xferAscUDP(c.conn, frame)
	}
	return xferAsc(c.conn, frame)
}

// Close closes the connection.
func (c *Client3E) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client3E) validate(device string, start, count int) (string, error) {
	dev := strings.ToUpper(device)
	if _, ok := binCode[dev]; !ok {
		return "", fmt.Errorf("unsupported device %q", device)
	}
	if start < 0 {
		return "", fmt.Errorf("start must be >= 0, got %d", start)
	}
	if count <= 0 {
		return "", fmt.Errorf("count must be > 0, got %d", count)
	}
	return dev, nil
}

func (c *Client3E) binBody(dev string, start, count int) []byte {
	b := make([]byte, 6)
	copy(b, addrBin(dev, start))
	binary.LittleEndian.PutUint16(b[4:], uint16(count))
	return b
}

// ReadWords reads count word values from device starting at address start.
func (c *Client3E) ReadWords(device string, start, count int) ([]uint16, error) {
	dev, err := c.validate(device, start, count)
	if err != nil {
		return nil, err
	}
	if c.mode == ModeBinary {
		resp, err := c.sendBin(buildBin(c.timer, cmdRead, subcWord, c.binBody(dev, start, count)))
		if err != nil {
			return nil, err
		}
		raw, err := chkBin(resp)
		if err != nil {
			return nil, err
		}
		if len(raw) < count*2 {
			return nil, connErr(fmt.Sprintf("short payload: expected %d bytes, got %d", count*2, len(raw)))
		}
		vals := make([]uint16, count)
		for i := range vals {
			vals[i] = binary.LittleEndian.Uint16(raw[i*2:])
		}
		return vals, nil
	}
	body := addrAsc(dev, start) + fmt.Sprintf("%04X", count)
	resp, err := c.sendAsc(buildAsc(c.timer, cmdRead, subcWord, body))
	if err != nil {
		return nil, err
	}
	raw, err := chkAsc(resp)
	if err != nil {
		return nil, err
	}
	if len(raw) < count*4 {
		return nil, connErr(fmt.Sprintf("short payload: expected %d chars, got %d", count*4, len(raw)))
	}
	vals := make([]uint16, count)
	for i := range vals {
		v, err := strconv.ParseUint(raw[i*4:(i+1)*4], 16, 16)
		if err != nil {
			return nil, connErr(fmt.Sprintf("invalid word at index %d: %v", i, err))
		}
		vals[i] = uint16(v)
	}
	return vals, nil
}

// WriteWords writes values to device starting at address start.
func (c *Client3E) WriteWords(device string, start int, values []uint16) error {
	dev, err := c.validate(device, start, len(values))
	if err != nil {
		return err
	}
	if c.mode == ModeBinary {
		wbuf := make([]byte, len(values)*2)
		for i, v := range values {
			binary.LittleEndian.PutUint16(wbuf[i*2:], v)
		}
		body := append(c.binBody(dev, start, len(values)), wbuf...)
		resp, err := c.sendBin(buildBin(c.timer, cmdWrite, subcWord, body))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	body := addrAsc(dev, start) + fmt.Sprintf("%04X", len(values))
	for _, v := range values {
		body += fmt.Sprintf("%04X", v)
	}
	resp, err := c.sendAsc(buildAsc(c.timer, cmdWrite, subcWord, body))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}

// ReadBits reads count bit values from device starting at address start.
func (c *Client3E) ReadBits(device string, start, count int) ([]bool, error) {
	dev, err := c.validate(device, start, count)
	if err != nil {
		return nil, err
	}
	if c.mode == ModeBinary {
		resp, err := c.sendBin(buildBin(c.timer, cmdRead, subcBit, c.binBody(dev, start, count)))
		if err != nil {
			return nil, err
		}
		raw, err := chkBin(resp)
		if err != nil {
			return nil, err
		}
		if expected := (count + 1) / 2; len(raw) < expected {
			return nil, connErr(fmt.Sprintf("short payload: expected %d bytes, got %d", expected, len(raw)))
		}
		bits := make([]bool, count)
		for i := range bits {
			b := raw[i/2]
			if i%2 == 0 {
				bits[i] = (b>>4)&0x01 != 0
			} else {
				bits[i] = b&0x01 != 0
			}
		}
		return bits, nil
	}
	body := addrAsc(dev, start) + fmt.Sprintf("%04X", count)
	resp, err := c.sendAsc(buildAsc(c.timer, cmdRead, subcBit, body))
	if err != nil {
		return nil, err
	}
	raw, err := chkAsc(resp)
	if err != nil {
		return nil, err
	}
	if len(raw) < count {
		return nil, connErr(fmt.Sprintf("short payload: expected %d chars, got %d", count, len(raw)))
	}
	bits := make([]bool, count)
	for i := range bits {
		bits[i] = raw[i] == '1'
	}
	return bits, nil
}

// WriteBits writes bit values to device starting at address start.
func (c *Client3E) WriteBits(device string, start int, values []bool) error {
	dev, err := c.validate(device, start, len(values))
	if err != nil {
		return err
	}
	if c.mode == ModeBinary {
		buf := make([]byte, (len(values)+1)/2)
		for i, v := range values {
			if v {
				if i%2 == 0 {
					buf[i/2] |= 0x10
				} else {
					buf[i/2] |= 0x01
				}
			}
		}
		body := append(c.binBody(dev, start, len(values)), buf...)
		resp, err := c.sendBin(buildBin(c.timer, cmdWrite, subcBit, body))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	body := addrAsc(dev, start) + fmt.Sprintf("%04X", len(values))
	for _, v := range values {
		if v {
			body += "1"
		} else {
			body += "0"
		}
	}
	resp, err := c.sendAsc(buildAsc(c.timer, cmdWrite, subcBit, body))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}
