package gomcprotocol

import (
	"fmt"
	"net"
	"time"
)

const maxTestRequestDataLen = 1024

func assertNoRequestBytes(conn net.Conn, wait time.Duration) error {
	if err := conn.SetReadDeadline(time.Now().Add(wait)); err != nil {
		return err
	}
	var b [1]byte
	n, err := conn.Read(b[:])
	clearErr := conn.SetReadDeadline(time.Time{})
	if n > 0 || err == nil {
		return fmt.Errorf("request byte arrived before first response")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		return err
	}
	return clearErr
}
