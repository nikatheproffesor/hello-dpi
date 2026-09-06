package dpi

import (
	"fmt"
	"net"
	"time"
)

// FakePacketStrategy sends a decoy / fake packet with low IP TTL (hop limit)
// to desynchronize stateful DPI inspection tables (Sandvine, Procera, Huawei)
// while ensuring the fake packet expires in transit before reaching the real destination server.
type FakePacketStrategy struct {
	decoyLen int
	fakeTTL  int
	delay    time.Duration
}

// NewFakePacketStrategy creates a fake/decoy packet desync strategy with low-level socket TTL control.
func NewFakePacketStrategy(decoyLen int, delayMs int) *FakePacketStrategy {
	if decoyLen <= 0 {
		decoyLen = 24
	}
	if delayMs <= 0 {
		delayMs = 5
	}
	return &FakePacketStrategy{
		decoyLen: decoyLen,
		fakeTTL:  3, // Low TTL ensures packet reaches ISP DPI but expires before server
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

	// 1. Set low TTL so the fake packet only reaches ISP DPI middlebox
	_ = SetSocketTTL(conn, s.fakeTTL)

	// 2. Transmit decoy segment
	var decoy []byte
	if info.Type == TypeTLSClientHello {
		decoy = GenerateDecoyTLSRecord(s.decoyLen)
	} else {
		decoy = []byte("GET /hello-dpi-probe HTTP/1.1\r\nHost: cdn.dummy.internal\r\n\r\n")
	}

	if _, err := conn.Write(decoy); err != nil {
		// If low TTL write fails, restore TTL and fallback
		_ = SetSocketTTL(conn, 64)
		return fmt.Errorf("decoy packet write failed: %w", err)
	}

	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	// 3. Restore standard TTL (64) for real payload
	_ = SetSocketTTL(conn, 64)

	// 4. Transmit real payload with 2-byte TLS record split
	tlsStrat := NewTLSRecordSplitStrategy(2, int(s.delay.Milliseconds()))
	return tlsStrat.Apply(conn, data, info)
}

func init() {
	RegisterStrategy(NewFakePacketStrategy(24, 5))
}
