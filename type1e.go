package gomcprotocol

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

// Client1E is a 1E frame MC Protocol TCP client.
// Targets Q series / L series CPUs in 1E-frame compatible access mode.
//
// Frame layout (binary, request):
//   [subheader 1B][PC No. 1B=0xFF][timer 2B LE][addr 3B LE][dev code 1B][count 2B LE][data]
//
// Frame layout (binary, response):
//   [end code 2B LE][data]
//
// ⚠ Bit operation subheader codes (sub1EBitRead/sub1EBitWrite) have not been
// verified on physical hardware. Word operations follow the commonly cited
// Q-series 1E frame specification.
type Client1E struct {
	host    string
	port    int
	timeout time.Duration
	timer   uint16
	conn    net.Conn
}

// 1E frame subheader codes (Q series CPU compatible).
const (
	sub1EWordRead  byte = 0x01
	sub1EWordWrite byte = 0x03
	sub1EBitRead   byte = 0x00 // ⚠ unverified
	sub1EBitWrite  byte = 0x02 // ⚠ unverified
)

// New1EClient creates a new 1E frame client. Call Connect before use.
func New1EClient(host string, port int) *Client1E {
	return &Client1E{
		host:    host,
		port:    port,
		timeout: 5 * time.Second,
		timer:   0x0010,
	}
}

// Connect establishes the TCP connection to the PLC.
func (c *Client1E) Connect() error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.host, c.port), c.timeout)
	if err != nil {
		return &MCProtocolConnectionError{msg: "connect: " + err.Error()}
	}
	c.conn = conn
	return nil
}

// Close closes the TCP connection.
func (c *Client1E) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client1E) validate(device string, start, count int) (string, error) {
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

// build1E constructs a 1E frame request.
func (c *Client1E) build1E(subheader byte, body []byte) []byte {
	frame := make([]byte, 4+len(body))
	frame[0] = subheader
	frame[1] = 0xFF // PC No.
	frame[2] = byte(c.timer)
	frame[3] = byte(c.timer >> 8)
	copy(frame[4:], body)
	return frame
}

// xfer1E sends a frame and receives the 1E response.
func (c *Client1E) xfer1E(frame []byte) ([]byte, error) {
	if _, err := c.conn.Write(frame); err != nil {
		return nil, connErr("send: " + err.Error())
	}
	// Read until connection closes; 1E has no explicit length field in the response header.
	buf := make([]byte, 4096)
	n, err := c.conn.Read(buf)
	if err != nil && n == 0 {
		return nil, connErr("recv: " + err.Error())
	}
	return buf[:n], nil
}

func chk1E(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, connErr(fmt.Sprintf("short response (%d bytes)", len(data)))
	}
	if ec := binary.LittleEndian.Uint16(data); ec != 0 {
		return nil, &MCProtocolError{EndCode: ec}
	}
	return data[2:], nil
}

func body1E(dev string, start, count int) []byte {
	b := make([]byte, 6)
	copy(b, addrBin(dev, start)) // 3 bytes addr + 1 byte device code
	binary.LittleEndian.PutUint16(b[4:], uint16(count))
	return b
}

// ReadWords reads count word values from device starting at address start.
func (c *Client1E) ReadWords(device string, start, count int) ([]uint16, error) {
	dev, err := c.validate(device, start, count)
	if err != nil {
		return nil, err
	}
	resp, err := c.xfer1E(c.build1E(sub1EWordRead, body1E(dev, start, count)))
	if err != nil {
		return nil, err
	}
	raw, err := chk1E(resp)
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

// WriteWords writes values to device starting at address start.
func (c *Client1E) WriteWords(device string, start int, values []uint16) error {
	dev, err := c.validate(device, start, len(values))
	if err != nil {
		return err
	}
	body := body1E(dev, start, len(values))
	wbuf := make([]byte, len(values)*2)
	for i, v := range values {
		binary.LittleEndian.PutUint16(wbuf[i*2:], v)
	}
	resp, err := c.xfer1E(c.build1E(sub1EWordWrite, append(body, wbuf...)))
	if err != nil {
		return err
	}
	_, err = chk1E(resp)
	return err
}

// ReadBits reads count bit values from device starting at address start.
// Each response byte represents one bit point (0x00=OFF, non-zero=ON).
// ⚠ Subheader code unverified on physical hardware.
func (c *Client1E) ReadBits(device string, start, count int) ([]bool, error) {
	dev, err := c.validate(device, start, count)
	if err != nil {
		return nil, err
	}
	resp, err := c.xfer1E(c.build1E(sub1EBitRead, body1E(dev, start, count)))
	if err != nil {
		return nil, err
	}
	raw, err := chk1E(resp)
	if err != nil {
		return nil, err
	}
	if len(raw) < count {
		return nil, connErr(fmt.Sprintf("short payload: expected %d bytes, got %d", count, len(raw)))
	}
	bits := make([]bool, count)
	for i := range bits {
		bits[i] = raw[i] != 0x00
	}
	return bits, nil
}

// WriteBits writes bit values to device starting at address start.
// Each bit is sent as one byte (0x00=OFF, 0x01=ON).
// ⚠ Subheader code unverified on physical hardware.
func (c *Client1E) WriteBits(device string, start int, values []bool) error {
	dev, err := c.validate(device, start, len(values))
	if err != nil {
		return err
	}
	body := body1E(dev, start, len(values))
	buf := make([]byte, len(values))
	for i, v := range values {
		if v {
			buf[i] = 0x01
		}
	}
	resp, err := c.xfer1E(c.build1E(sub1EBitWrite, append(body, buf...)))
	if err != nil {
		return err
	}
	_, err = chk1E(resp)
	return err
}

