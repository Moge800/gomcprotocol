// Package gomcprotocol implements a Mitsubishi MC Protocol (3E frame) TCP client.
// Inspired by pymcprotocol and micromcprotocol.
package gomcprotocol

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Client3E is a 3E frame MC Protocol TCP client.
type Client3E struct {
	host    string
	port    int
	mode    Mode
	timeout time.Duration
	timer   uint16
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

// Connect establishes the TCP connection to the PLC.
func (c *Client3E) Connect() error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return &MCProtocolConnectionError{msg: "connect: " + err.Error()}
	}
	c.conn = conn
	return nil
}

// Close closes the TCP connection.
func (c *Client3E) Close() error {
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
		resp, err := xferBin(c.conn, buildBin(c.timer, cmdRead, subcWord, c.binBody(dev, start, count)))
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
	resp, err := xferAsc(c.conn, buildAsc(c.timer, cmdRead, subcWord, body))
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
		v, _ := strconv.ParseUint(raw[i*4:(i+1)*4], 16, 16)
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
		resp, err := xferBin(c.conn, buildBin(c.timer, cmdWrite, subcWord, body))
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
	resp, err := xferAsc(c.conn, buildAsc(c.timer, cmdWrite, subcWord, body))
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
		resp, err := xferBin(c.conn, buildBin(c.timer, cmdRead, subcBit, c.binBody(dev, start, count)))
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
	resp, err := xferAsc(c.conn, buildAsc(c.timer, cmdRead, subcBit, body))
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
		resp, err := xferBin(c.conn, buildBin(c.timer, cmdWrite, subcBit, body))
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
	resp, err := xferAsc(c.conn, buildAsc(c.timer, cmdWrite, subcBit, body))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}
