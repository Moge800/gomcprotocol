package gomcprotocol

import "fmt"

// RemoteRun starts the PLC CPU remotely (command 0x1001).
//
//   clearMode: 0 = no clear, 1 = clear except latch, 2 = clear all
//   force:     true to execute even if another device is controlling remotely
func (c *Client3E) RemoteRun(clearMode int, force bool) error {
	if clearMode < 0 || clearMode > 2 {
		return fmt.Errorf("clearMode must be 0, 1, or 2")
	}
	mode := uint16(0x0001)
	if force {
		mode = 0x0003
	}
	if c.mode == ModeBinary {
		body := []byte{
			byte(mode), byte(mode >> 8),
			byte(clearMode), 0x00,
		}
		resp, err := c.sendBin(buildBin(c.timer, 0x1001, 0x0000, body))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	body := fmt.Sprintf("%04X%02X00", mode, clearMode)
	resp, err := c.sendAsc(buildAsc(c.timer, 0x1001, 0x0000, body))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}

// RemoteStop stops the PLC CPU remotely (command 0x1002).
func (c *Client3E) RemoteStop() error {
	if c.mode == ModeBinary {
		resp, err := c.sendBin(buildBin(c.timer, 0x1002, 0x0000, []byte{0x01, 0x00}))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	resp, err := c.sendAsc(buildAsc(c.timer, 0x1002, 0x0000, "0001"))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}

// RemotePause pauses the PLC CPU remotely (command 0x1003).
func (c *Client3E) RemotePause(force bool) error {
	mode := uint16(0x0001)
	if force {
		mode = 0x0003
	}
	if c.mode == ModeBinary {
		body := []byte{byte(mode), byte(mode >> 8)}
		resp, err := c.sendBin(buildBin(c.timer, 0x1003, 0x0000, body))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	resp, err := c.sendAsc(buildAsc(c.timer, 0x1003, 0x0000, fmt.Sprintf("%04X", mode)))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}

// RemoteLatchClear clears the latch remotely (command 0x1005).
// The PLC must be stopped before calling this.
func (c *Client3E) RemoteLatchClear() error {
	if c.mode == ModeBinary {
		resp, err := c.sendBin(buildBin(c.timer, 0x1005, 0x0000, []byte{0x01, 0x00}))
		if err != nil {
			return err
		}
		_, err = chkBin(resp)
		return err
	}
	resp, err := c.sendAsc(buildAsc(c.timer, 0x1005, 0x0000, "0001"))
	if err != nil {
		return err
	}
	_, err = chkAsc(resp)
	return err
}

// RemoteReset resets the PLC remotely (command 0x1006).
// The PLC must be stopped before calling this.
// The connection may be closed by the PLC before a response is received.
func (c *Client3E) RemoteReset() error {
	if c.mode == ModeBinary {
		c.sendBin(buildBin(c.timer, 0x1006, 0x0000, []byte{0x01, 0x00})) //nolint
		c.Close()
		return nil
	}
	c.sendAsc(buildAsc(c.timer, 0x1006, 0x0000, "0001")) //nolint
	c.Close()
	return nil
}
