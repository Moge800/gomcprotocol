package gomcprotocol

import "fmt"

// MCProtocolError is returned when the PLC responds with a non-zero end code.
type MCProtocolError struct {
	EndCode uint16
}

func (e *MCProtocolError) Error() string {
	return fmt.Sprintf("MC error 0x%04X", e.EndCode)
}

// MCProtocolConnectionError is returned on network-level failures.
type MCProtocolConnectionError struct {
	msg string
}

func (e *MCProtocolConnectionError) Error() string {
	return e.msg
}

func connErr(msg string) error {
	return &MCProtocolConnectionError{msg: msg}
}
