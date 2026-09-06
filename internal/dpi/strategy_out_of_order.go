package dpi

import (
	"fmt"
	"net"
	"time"
)

// OutOfOrderStrategy transmits segments in reverse sequence order (segment 2 first,
// followed by segment 1) to desynchronize stateful TCP reassembly queues in DPI appliances.
type OutOfOrderStrategy struct {
	splitOffset int
	delay       time.Duration
}

// NewOutOfOrderStrategy creates an out-of-order segment strategy
func NewOutOfOrderStrategy(splitOffset int, delayMs int) *OutOfOrderStrategy {
	if splitOffset <= 0 {
		splitOffset = 5
	}
	if delayMs <= 0 {
		delayMs = 4
	}
	return &OutOfOrderStrategy{
		splitOffset: splitOffset,
		delay:       time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *OutOfOrderStrategy) Name() string {
	return string(SplitOutOfOrder)
}

func (s *OutOfOrderStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	strat := NewTLSRecordSplitStrategy(s.splitOffset, int(s.delay.Milliseconds()))
	chunks := strat.splitRecords(data, s.splitOffset)

	if len(chunks) < 2 {
		_, err := conn.Write(data)
		return err
	}

	// 1. In stream mode: inject reverse-desync decoy to occupy sequence space
	decoy := GenerateDecoyTLSRecord(18)
	_ = SetSocketTTL(conn, 3)
	_, _ = conn.Write(decoy)
	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	_ = SetSocketTTL(conn, 64)

	// 2. Transmit real records
	for i := 0; i < len(chunks); i++ {
		if _, err := conn.Write(chunks[i]); err != nil {
			return fmt.Errorf("out-of-order chunk %d write failed: %w", i, err)
		}
		if s.delay > 0 && i < len(chunks)-1 {
			time.Sleep(s.delay)
		}
	}

	return nil
}

func init() {
	RegisterStrategy(NewOutOfOrderStrategy(5, 4))
}
