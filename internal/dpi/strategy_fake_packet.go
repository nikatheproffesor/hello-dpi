package dpi

import (
	"fmt"
	"net"
	"time"
)

// FakePacketStrategy sends a decoy / fake packet (such as a dummy TLS record or
// malformed alert) to corrupt and desynchronize the stateful DPI inspection window
// before transmitting the real payload.
type FakePacketStrategy struct {
	decoyLen int
	delay    time.Duration
}

// NewFakePacketStrategy creates a fake/decoy packet desync strategy.
func NewFakePacketStrategy(decoyLen int, delayMs int) *FakePacketStrategy {
	if decoyLen <= 0 {
		decoyLen = 24
	}
	if delayMs <= 0 {
		delayMs = 5
	}
	return &FakePacketStrategy{
		decoyLen: decoyLen,
		delay:    time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *FakePacketStrategy) Name() string {
	return string(SplitDecoy)
}

func (s *FakePacketStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	// 1. Send decoy segment to trick stateful DPI reassembler
	var decoy []byte
	if info.Type == TypeTLSClientHello {
		decoy = GenerateDecoyTLSRecord(s.decoyLen)
	} else {
		// HTTP decoy request line
		decoy = []byte("GET /hello-dpi-probe HTTP/1.1\r\nHost: cdn.dummy.internal\r\n\r\n")
	}

	if _, err := conn.Write(decoy); err != nil {
		return fmt.Errorf("decoy packet write failed: %w", err)
	}

	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	// 2. Transmit real payload using 2-byte TLS record split or first-byte
	tlsStrat := NewTLSRecordSplitStrategy(2, int(s.delay.Milliseconds()))
	return tlsStrat.Apply(conn, data, info)
}

func init() {
	RegisterStrategy(NewFakePacketStrategy(24, 5))
}
