package gomcprotocol

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// DeviceAddr identifies a single device point for random-access operations.
type DeviceAddr struct {
	Device string
	Addr   int
}

func (c *Client3E) validateAddrs(devices []DeviceAddr) ([]string, error) {
	devs := make([]string, len(devices))
	for i, d := range devices {
		dev, err := c.validate(d.Device, d.Addr, 1)
		if err != nil {
			return nil, fmt.Errorf("device[%d]: %w", i, err)
		}
		devs[i] = dev
	}
	return devs, nil
}

// RandomRead reads word (2-byte) and dword (4-byte) values from multiple
// devices in a single request (command 0x0403).
func (c *Client3E) RandomRead(words, dwords []DeviceAddr) ([]uint16, []uint32, error) {
	if len(words) > 255 || len(dwords) > 255 {
		return nil, nil, fmt.Errorf("device count must be <= 255")
	}
	wDevs, err := c.validateAddrs(words)
	if err != nil {
		return nil, nil, err
	}
	dDevs, err := c.validateAddrs(dwords)
	if err != nil {
		return nil, nil, err
	}
	if c.mode == ModeBinary {
		body := make([]byte, 0, 2+4*(len(words)+len(dwords)))
		body = append(body, byte(len(words)), byte(len(dwords)))
		for i, d := range words {
			body = append(body, addrBin(wDevs[i], d.Addr)...)
		}
		for i, d := range dwords {
			body = append(body, addrBin(dDevs[i], d.Addr)...)
		}
		resp, err := c.sendBin(buildBin(c.timer, 0x0403, 0x0000, body))
		if err != nil {
			return nil, nil, err
		}
		raw, err := chkBin(resp)
		if err != nil {
			return nil, nil, err
		}
		expected := len(words)*2 + len(dwords)*4
		if len(raw) < expected {
			return nil, nil, connErr(fmt.Sprintf("short payload: expected %d bytes, got %d", expected, len(raw)))
		}
		wVals := make([]uint16, len(words))
		for i := range wVals {
			wVals[i] = binary.LittleEndian.Uint16(raw[i*2:])
		}
		dVals := make([]uint32, len(dwords))
		off := len(words) * 2
		for i := range dVals {
			dVals[i] = binary.LittleEndian.Uint32(raw[off+i*4:])
		}
		return wVals, dVals, nil
	}
	// ASCII mode
	body := fmt.Sprintf("%02X%02X", len(words), len(dwords))
	for i, d := range words {
		body += addrAsc(wDevs[i], d.Addr)
	}
	for i, d := range dwords {
		body += addrAsc(dDevs[i], d.Addr)
	}
	resp, err := c.sendAsc(buildAsc(c.timer, 0x0403, 0x0000, body))
	if err != nil {
		return nil, nil, err
	}
	raw, err := chkAsc(resp)
	if err != nil {
		return nil, nil, err
	}
	if expected := len(words)*4 + len(dwords)*8; len(raw) < expected {
		return nil, nil, connErr(fmt.Sprintf("short payload: expected %d chars, got %d", expected, len(raw)))
	}
	wVals := make([]uint16, len(words))
	for i := range wVals {
		v, err := strconv.ParseUint(raw[i*4:(i+1)*4], 16, 16)
		if err != nil {
			return nil, nil, connErr(fmt.Sprintf("invalid word at index %d: %v", i, err))
		}
		wVals[i] = uint16(v)
	}
	dVals := make([]uint32, len(dwords))
	off := len(words) * 4
	for i := range dVals {
		v, err := strconv.ParseUint(raw[off+i*8:off+(i+1)*8], 16, 32)
		if err != nil {
			return nil, nil, connErr(fmt.Sprintf("invalid dword at index %d: %v", i, err))
		}
		dVals[i] = uint32(v)
	}
	return wVals, dVals, nil
}

// RandomWrite writes word (2-byte) and dword (4-byte) values to multiple
// devices in a single request (command 0x1402, subcmd 0x0000).
func (c *Client3E) RandomWrite(words []DeviceAddr, wordVals []uint16, dwords []DeviceAddr, dwordVals []uint32) error {
	if len(words) != len(wordVals) {
		return fmt.Errorf("words and wordVals must be same length")
	}
	if len(dwords) != len(dwordVals) {
		return fmt.Errorf("dwords and dwordVals must be same length")
	}
	if len(words) > 255 || len(dwords) > 255 {
		return fmt.Errorf("device count must be <= 255")
	}
	wDevs, err := c.validateAddrs(words)
	if err != nil {
		return err
	}
	dDevs, err := c.validateAddrs(dwords)
	if err != nil {
		return err
	}
	if c.mode == ModeBinary {
		body := make([]byte, 0, 2+6*len(words)+8*len(dwords))
		body = append(body, byte(len(words)), byte(len(dwords)))
		for i, d := range words {
			body = append(body, addrBin(wDevs[i], d.Addr)...)
			body = append(body, byte(wordVals[i]), byte(wordVals[i]>>8))
		}
		for i, d := range dwords {
			body = append(body, addrBin(dDevs[i], d.Addr)...)
			v := dwordVals[i]
			body = append(body, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
		}
		resp, err := c.sendBin(buildBin(c.timer, 0x1402, 0x0000, body))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	// ASCII mode
	body := fmt.Sprintf("%02X%02X", len(words), len(dwords))
	for i, d := range words {
		body += addrAsc(wDevs[i], d.Addr) + fmt.Sprintf("%04X", wordVals[i])
	}
	for i, d := range dwords {
		body += addrAsc(dDevs[i], d.Addr) + fmt.Sprintf("%08X", dwordVals[i])
	}
	resp, err := c.sendAsc(buildAsc(c.timer, 0x1402, 0x0000, body))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}

// RandomWriteBits writes individual bit values to multiple devices in a
// single request (command 0x1402, subcmd 0x0001).
func (c *Client3E) RandomWriteBits(devices []DeviceAddr, values []bool) error {
	if len(devices) != len(values) {
		return fmt.Errorf("devices and values must be same length")
	}
	if len(devices) > 255 {
		return fmt.Errorf("device count must be <= 255")
	}
	devs, err := c.validateAddrs(devices)
	if err != nil {
		return err
	}
	if c.mode == ModeBinary {
		body := make([]byte, 0, 1+5*len(devices))
		body = append(body, byte(len(devices)))
		for i, d := range devices {
			body = append(body, addrBin(devs[i], d.Addr)...)
			if values[i] {
				body = append(body, 0x01)
			} else {
				body = append(body, 0x00)
			}
		}
		resp, err := c.sendBin(buildBin(c.timer, 0x1402, 0x0001, body))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	// ASCII mode
	body := fmt.Sprintf("%02X", len(devices))
	for i, d := range devices {
		body += addrAsc(devs[i], d.Addr)
		if values[i] {
			body += "01"
		} else {
			body += "00"
		}
	}
	resp, err := c.sendAsc(buildAsc(c.timer, 0x1402, 0x0001, body))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}
