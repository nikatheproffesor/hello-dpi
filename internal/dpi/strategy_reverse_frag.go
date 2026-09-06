package dpi

import (
	"fmt"
	"net"
	"time"
)

const (
	SplitReverseFrag SplitMode = "reverse-frag"
)

// ReverseFragStrategy transmits payload segments with reverse fragmentation or
// out-of-order timing simulation to defeat stateful TCP reassembly in DPI middleboxes.
type ReverseFragStrategy struct {
	splitOffset int
	delay       time.Duration
}

// NewReverseFragStrategy creates a new reverse fragmentation strategy.
func NewReverseFragStrategy(splitOffset int, delayMs int) *ReverseFragStrategy {
	if splitOffset <= 0 {
		splitOffset = 3
	}
	if delayMs <= 0 {
		delayMs = 4
	}
	return &ReverseFragStrategy{
		splitOffset: splitOffset,
		delay:       time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *ReverseFragStrategy) Name() string {
	return string(SplitReverseFrag)
}

func (s *ReverseFragStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	strat := NewTLSRecordSplitStrategy(s.splitOffset, int(s.delay.Milliseconds()))
	chunks := strat.splitRecords(data, s.splitOffset)

	if len(chunks) < 2 {
		_, err := conn.Write(data)
		return err
	}

	// Transmit first chunk (record header)
	if _, err := conn.Write(chunks[0]); err != nil {
		return fmt.Errorf("reverse-frag chunk 0 write failed: %w", err)
	}

	// Micro timing pause
	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	// Transmit remaining chunks with staggered pacing
	for i := 1; i < len(chunks); i++ {
		if _, err := conn.Write(chunks[i]); err != nil {
			return fmt.Errorf("reverse-frag chunk %d write failed: %w", i, err)
		}
		if s.delay > 0 && i < len(chunks)-1 {
			time.Sleep(s.delay / 2)
		}
	}

	return nil
}

func init() {
	RegisterStrategy(NewReverseFragStrategy(3, 4))
}


type aliasStrategy struct {
	name     string
	delegate BypassStrategy
}

func (a *aliasStrategy) Name() string {
	return a.name
}

func (a *aliasStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	return a.delegate.Apply(conn, data, info)
}
