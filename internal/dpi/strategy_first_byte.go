package dpi

import (
	"fmt"
	"net"
	"time"
)

// FirstByteSplitStrategy isolates the first byte (e.g. 0x16 TLS handshake byte)
// and sends it in an independent TCP packet before transmitting the remainder.
type FirstByteSplitStrategy struct {
	delay time.Duration
}

// NewFirstByteSplitStrategy creates a first-byte split strategy.
func NewFirstByteSplitStrategy(delayMs int) *FirstByteSplitStrategy {
	if delayMs <= 0 {
		delayMs = 3
	}
	return &FirstByteSplitStrategy{
		delay: time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *FirstByteSplitStrategy) Name() string {
	return string(SplitFirstByte)
}

func (s *FirstByteSplitStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	if len(data) <= 1 {
		_, err := conn.Write(data)
		return err
	}

	if _, err := conn.Write(data[:1]); err != nil {
		return fmt.Errorf("first-byte chunk 0 write failed: %w", err)
	}

	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	if _, err := conn.Write(data[1:]); err != nil {
		return fmt.Errorf("first-byte chunk 1 write failed: %w", err)
	}

	return nil
}

func init() {
	RegisterStrategy(NewFirstByteSplitStrategy(3))
}
