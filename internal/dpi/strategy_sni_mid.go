package dpi

import (
	"fmt"
	"net"
	"time"
)

// SNIMidSplitStrategy identifies the exact SNI hostname location inside ClientHello
// and splits the packet directly in the middle of the hostname with a timing delay.
type SNIMidSplitStrategy struct {
	delay time.Duration
}

// NewSNIMidSplitStrategy creates an SNI-mid split strategy with the specified delay.
func NewSNIMidSplitStrategy(delayMs int) *SNIMidSplitStrategy {
	if delayMs <= 0 {
		delayMs = 5
	}
	return &SNIMidSplitStrategy{
		delay: time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *SNIMidSplitStrategy) Name() string {
	return string(SplitSNI)
}

func (s *SNIMidSplitStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	splitPoint := -1
	if info.Type == TypeTLSClientHello && info.SNIOffset > 0 && info.SNILength > 1 {
		// Split right in the middle of the SNI hostname
		splitPoint = info.SNIOffset + (info.SNILength / 2)
	}

	// Fallback to record-based or byte-level mid split
	if splitPoint <= 0 || splitPoint >= len(data) {
		if len(data) > 10 {
			splitPoint = len(data) / 2
		} else if len(data) > 1 {
			splitPoint = 1
		} else {
			_, err := conn.Write(data)
			return err
		}
	}

	part1 := data[:splitPoint]
	part2 := data[splitPoint:]

	if _, err := conn.Write(part1); err != nil {
		return fmt.Errorf("sni-mid part 1 write failed: %w", err)
	}

	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	if _, err := conn.Write(part2); err != nil {
		return fmt.Errorf("sni-mid part 2 write failed: %w", err)
	}

	return nil
}

func init() {
	RegisterStrategy(NewSNIMidSplitStrategy(5))
}
