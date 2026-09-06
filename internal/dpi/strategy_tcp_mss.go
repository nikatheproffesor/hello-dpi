package dpi

import (
	"fmt"
	"net"
	"time"
)

const (
	SplitTCPMSS SplitMode = "tcp-mss"
)

// TCPWindowMSSStrategy forces TCP Maximum Segment Size (MSS) manipulation at the socket layer.
// By constraining MSS to a small value (e.g. 64-128 bytes), the kernel TCP stack automatically
// slices the initial TLS ClientHello into micro-packets across the network interface.
type TCPWindowMSSStrategy struct {
	mss   int
	delay time.Duration
}

// NewTCPWindowMSSStrategy creates a TCP MSS manipulation strategy
func NewTCPWindowMSSStrategy(mss int, delayMs int) *TCPWindowMSSStrategy {
	if mss <= 0 {
		mss = 88 // 88 bytes guarantees ClientHello is divided across 3+ packets
	}
	if delayMs <= 0 {
		delayMs = 2
	}
	return &TCPWindowMSSStrategy{
		mss:   mss,
		delay: time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *TCPWindowMSSStrategy) Name() string {
	return string(SplitTCPMSS)
}

func (s *TCPWindowMSSStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	// 1. Attempt kernel socket option TCP_MAXSEG
	err := SetSocketMSS(conn, s.mss)
	if err == nil {
		// Native kernel MSS successfully set; writing payload will be automatically segmented by OS
		_, wErr := conn.Write(data)
		return wErr
	}

	// 2. Fallback: software micro-chunking to simulate MSS
	for i := 0; i < len(data); i += s.mss {
		end := i + s.mss
		if end > len(data) {
			end = len(data)
		}
		if _, err := conn.Write(data[i:end]); err != nil {
			return fmt.Errorf("tcp-mss chunk write failed: %w", err)
		}
		if end < len(data) && s.delay > 0 {
			time.Sleep(s.delay)
		}
	}

	return nil
}

func init() {
	RegisterStrategy(NewTCPWindowMSSStrategy(88, 2))
}
