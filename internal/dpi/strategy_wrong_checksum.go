package dpi

import (
	"fmt"
	"net"
	"time"
)

const (
	SplitWrongChecksum SplitMode = "wrong-chksum"
)

// WrongChecksumStrategy exploits hardware DPI middleboxes that skip L4 checksum calculation
// to save CPU cycles. A decoy packet is injected with wrong checksum parameters or malformed
// L4 payload. The destination server's NIC kernel drops it, while the DPI appliance processes it
// and becomes desynchronized.
type WrongChecksumStrategy struct {
	fakeTTL int
	delay   time.Duration
}

// NewWrongChecksumStrategy creates a new wrong checksum strategy
func NewWrongChecksumStrategy(fakeTTL int, delayMs int) *WrongChecksumStrategy {
	if fakeTTL <= 0 {
		fakeTTL = 3
	}
	if delayMs <= 0 {
		delayMs = 4
	}
	return &WrongChecksumStrategy{
		fakeTTL: fakeTTL,
		delay:   time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *WrongChecksumStrategy) Name() string {
	return string(SplitWrongChecksum)
}

func (s *WrongChecksumStrategy) SendDecoy(conn net.Conn, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	// Set low TTL so corrupted payload expires before destination
	_ = SetSocketTTL(conn, s.fakeTTL)

	// Send decoy corrupt payload (dummy TLS Handshake record with invalid length checksum)
	badDecoy := []byte{0x16, 0x03, 0x01, 0xFF, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}
	_, err := conn.Write(badDecoy)
	_ = SetSocketTTL(conn, 64)
	if err != nil {
		return fmt.Errorf("wrong-chksum write failed: %w", err)
	}

	if s.delay > 0 {
		time.Sleep(s.delay)
	}
	return nil
}

func (s *WrongChecksumStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if err := s.SendDecoy(conn, info); err != nil {
		return err
	}

	// Transmit real payload with 5-byte RFC record split
	tlsStrat := NewTLSRecordSplitStrategy(5, int(s.delay.Milliseconds()))
	return tlsStrat.Apply(conn, data, info)
}

func init() {
	RegisterStrategy(NewWrongChecksumStrategy(3, 4))
}
