package gomcprotocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

var binHdr = []byte{0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00}

const ascHdr = "500000FF03FF00"

func buildBin(timer, cmd, subcmd uint16, body []byte) []byte {
	payload := make([]byte, 6+len(body))
	binary.LittleEndian.PutUint16(payload[0:], timer)
	binary.LittleEndian.PutUint16(payload[2:], cmd)
	binary.LittleEndian.PutUint16(payload[4:], subcmd)
	copy(payload[6:], body)
	frame := make([]byte, 9+len(payload))
	copy(frame[:7], binHdr)
	binary.LittleEndian.PutUint16(frame[7:], uint16(len(payload)))
	copy(frame[9:], payload)
	return frame
}

func buildAsc(timer, cmd, subcmd uint16, body string) string {
	inner := fmt.Sprintf("%04X%04X%04X%s", timer, cmd, subcmd, body)
	return ascHdr + fmt.Sprintf("%04X", len(inner)) + inner
}

func xferBin(conn net.Conn, frame []byte) ([]byte, error) {
	if _, err := conn.Write(frame); err != nil {
		return nil, connErr("send: " + err.Error())
	}
	hdr := make([]byte, 9)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return nil, connErr("recv header: " + err.Error())
	}
	body := make([]byte, int(binary.LittleEndian.Uint16(hdr[7:])))
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, connErr("recv body: " + err.Error())
	}
	return append(hdr, body...), nil
}

func xferAsc(conn net.Conn, frame string) (string, error) {
	if _, err := conn.Write([]byte(frame)); err != nil {
		return "", connErr("send: " + err.Error())
	}
	hdr := make([]byte, 18)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return "", connErr("recv header: " + err.Error())
	}
	n, err := strconv.ParseUint(string(hdr[14:18]), 16, 16)
	if err != nil {
		return "", connErr("invalid response length")
	}
	body := make([]byte, int(n))
	if _, err := io.ReadFull(conn, body); err != nil {
		return "", connErr("recv body: " + err.Error())
	}
	return string(hdr) + string(body), nil
}

func chkBin(data []byte) ([]byte, error) {
	if len(data) < 11 {
		return nil, connErr(fmt.Sprintf("short response (%d bytes)", len(data)))
	}
	if ec := binary.LittleEndian.Uint16(data[9:]); ec != 0 {
		return nil, &MCProtocolError{EndCode: ec}
	}
	return data[11:], nil
}

func chkAsc(data string) (string, error) {
	if len(data) < 22 {
		return "", connErr(fmt.Sprintf("short response (%d chars)", len(data)))
	}
	ec, err := strconv.ParseUint(data[18:22], 16, 16)
	if err != nil {
		return "", connErr("invalid end code in response")
	}
	if ec != 0 {
		return "", &MCProtocolError{EndCode: uint16(ec)}
	}
	return data[22:], nil
}

// xferBinUDP sends a binary frame over UDP and reads the response datagram.
// The 4096-byte buffer handles typical responses; callers should prefer TCP
// for large reads whose response could exceed this size.
func xferBinUDP(conn net.Conn, frame []byte) ([]byte, error) {
	if _, err := conn.Write(frame); err != nil {
		return nil, connErr("send: " + err.Error())
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, connErr("recv: " + err.Error())
	}
	return buf[:n], nil
}

// xferAscUDP sends an ASCII frame over UDP and reads the response datagram.
// The 4096-byte buffer handles typical responses; callers should prefer TCP
// for large reads whose response could exceed this size.
func xferAscUDP(conn net.Conn, frame string) (string, error) {
	if _, err := conn.Write([]byte(frame)); err != nil {
		return "", connErr("send: " + err.Error())
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", connErr("recv: " + err.Error())
	}
	return string(buf[:n]), nil
}

// addrBin encodes a device address for binary mode.
// Layout: [addr_lo, addr_mid, addr_hi, device_code]
func addrBin(dev string, addr int) []byte {
	return []byte{byte(addr), byte(addr >> 8), byte(addr >> 16), binCode[dev]}
}

// addrAsc encodes a device address for ASCII mode.
// The address base follows the MC protocol device-code table.
func addrAsc(dev string, addr int) string {
	code := dev
	if mapped, ok := ascCode[dev]; ok {
		code = mapped
	}
	if decimalAddrDevs[dev] {
		return fmt.Sprintf("%-2s%06d", code, addr)
	}
	return fmt.Sprintf("%-2s%06X", code, addr)
}
