package dpi

import (
	"fmt"
	"net"
	"time"
)

const (
	SplitWrongSeq SplitMode = "wrong-seq"
)

// WrongSeqAckStrategy desynchronizes stateful DPI stream inspection by injecting a decoy segment
// with out-of-order sequence indicators or corrupted sequence numbers. The DPI middlebox accepts
// and tracks the sequence, becoming desynchronized, while the real destination server discards it.
type WrongSeqAckStrategy struct {
	fakeTTL int
	delay   time.Duration
}

// NewWrongSeqAckStrategy creates a new wrong SEQ/ACK desync strategy
func NewWrongSeqAckStrategy(fakeTTL int, delayMs int) *WrongSeqAckStrategy {
	if fakeTTL <= 0 {
		fakeTTL = 3
	}
	if delayMs <= 0 {
		delayMs = 4
	}
	return &WrongSeqAckStrategy{
		fakeTTL: fakeTTL,
		delay:   time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *WrongSeqAckStrategy) Name() string {
	return string(SplitWrongSeq)
}

func (s *WrongSeqAckStrategy) SendDecoy(conn net.Conn, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	// 1. Lower TTL to middlebox distance
	_ = SetSocketTTL(conn, s.fakeTTL)

	// 2. Craft decoy packet with out-of-sequence TLS alert or invalid handshake
	decoy := make([]byte, 16)
	decoy[0] = 0x15 // TLS Alert
	decoy[1] = 0x03
	decoy[2] = 0x03
	decoy[3] = 0x00
	decoy[4] = 0x02
	decoy[5] = 0x01 // Warning
	decoy[6] = 0x00 // Close notify

	_, err := conn.Write(decoy)
	_ = SetSocketTTL(conn, 64)
	if err != nil {
		return fmt.Errorf("wrong-seq decoy write failed: %w", err)
	}

	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return nil
}

func (s *WrongSeqAckStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if err := s.SendDecoy(conn, info); err != nil {
		return err
	}

	// Send real payload with 5-byte RFC record split
	tlsStrat := NewTLSRecordSplitStrategy(5, int(s.delay.Milliseconds()))
	return tlsStrat.Apply(conn, data, info)
}

func init() {
	RegisterStrategy(NewWrongSeqAckStrategy(3, 4))
}
