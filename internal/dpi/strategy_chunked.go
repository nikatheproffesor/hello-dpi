package dpi

import (
	"fmt"
	"net"
	"time"
)

// ChunkedSplitStrategy slices the payload into small micro-segments (e.g. 20-40 bytes)
// and transmits them with micro-timing gaps to thoroughly prevent DPI keyword matches.
type ChunkedSplitStrategy struct {
	chunkSize int
	delay     time.Duration
}

// NewChunkedSplitStrategy creates a chunked split strategy.
func NewChunkedSplitStrategy(chunkSize int, delayMs int) *ChunkedSplitStrategy {
	if chunkSize <= 0 {
		chunkSize = 32
	}
	if delayMs <= 0 {
		delayMs = 2
	}
	return &ChunkedSplitStrategy{
		chunkSize: chunkSize,
		delay:     time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *ChunkedSplitStrategy) Name() string {
	return string(SplitChunked)
}

func (s *ChunkedSplitStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	if len(data) <= s.chunkSize {
		_, err := conn.Write(data)
		return err
	}

	for i := 0; i < len(data); i += s.chunkSize {
		end := i + s.chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]

		if _, err := conn.Write(chunk); err != nil {
			return fmt.Errorf("chunked split write at offset %d failed: %w", i, err)
		}

		if end < len(data) && s.delay > 0 {
			time.Sleep(s.delay)
		}
	}

	return nil
}

func init() {
	RegisterStrategy(NewChunkedSplitStrategy(32, 2))
}
