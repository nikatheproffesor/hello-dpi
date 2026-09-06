package dpi

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

// HTTPTrickVariant specifies how the HTTP Host header is manipulated
type HTTPTrickVariant string

const (
	TrickHostCaseMix  HTTPTrickVariant = "case-mix"  // hoSt: example.com
	TrickHostSpacePre HTTPTrickVariant = "space-pre" // Host : example.com
	TrickHostTab      HTTPTrickVariant = "tab"       // Host:\texample.com
	TrickHostMidSplit HTTPTrickVariant = "mid-split" // Split across "Ho" and "st:"
)

// HTTPHostTrickStrategy implements HTTP Host header desynchronization to defeat
// DPI keyword filters on plain HTTP requests.
type HTTPHostTrickStrategy struct {
	variant HTTPTrickVariant
	delay   time.Duration
}

// NewHTTPHostTrickStrategy creates a new HTTP host trick strategy.
func NewHTTPHostTrickStrategy(variant HTTPTrickVariant, delayMs int) *HTTPHostTrickStrategy {
	if variant == "" {
		variant = TrickHostCaseMix
	}
	if delayMs <= 0 {
		delayMs = 5
	}
	return &HTTPHostTrickStrategy{
		variant: variant,
		delay:   time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *HTTPHostTrickStrategy) Name() string {
	return string(SplitHTTPMix)
}

func (s *HTTPHostTrickStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	if info.Type != TypeHTTPRequest || info.HostOffset < 0 || info.HostOffset+5 > len(data) {
		// Non-HTTP fallback: first byte split
		if len(data) <= 1 {
			_, err := conn.Write(data)
			return err
		}
		if _, err := conn.Write(data[:1]); err != nil {
			return err
		}
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
		_, err := conn.Write(data[1:])
		return err
	}

	modData := make([]byte, len(data))
	copy(modData, data)
	hOff := info.HostOffset

	switch s.variant {
	case TrickHostCaseMix:
		// Replace "host:" with "hoSt:"
		if bytes.Equal(bytes.ToLower(modData[hOff:hOff+5]), []byte("host:")) {
			modData[hOff] = 'h'
			modData[hOff+1] = 'o'
			modData[hOff+2] = 'S'
			modData[hOff+3] = 't'
		}
		// Also split right after "hoS"
		splitPt := hOff + 3
		return writeSplit(conn, modData, splitPt, s.delay)

	case TrickHostSpacePre:
		// Replace "Host:" with "Host :"
		var buf bytes.Buffer
		buf.Write(modData[:hOff+4])
		buf.WriteByte(' ')
		buf.Write(modData[hOff+4:])
		splitPt := hOff + 4
		return writeSplit(conn, buf.Bytes(), splitPt, s.delay)

	case TrickHostTab:
		// Replace space after colon with tab: "Host:\t..."
		if hOff+6 <= len(modData) && modData[hOff+5] == ' ' {
			modData[hOff+5] = '\t'
		}
		splitPt := hOff + 5
		return writeSplit(conn, modData, splitPt, s.delay)

	default: // TrickHostMidSplit
		splitPt := hOff + 2 // Between "Ho" and "st"
		return writeSplit(conn, modData, splitPt, s.delay)
	}
}

func writeSplit(conn net.Conn, data []byte, splitPt int, delay time.Duration) error {
	if splitPt <= 0 || splitPt >= len(data) {
		_, err := conn.Write(data)
		return err
	}

	if _, err := conn.Write(data[:splitPt]); err != nil {
		return fmt.Errorf("http trick part 1 write failed: %w", err)
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	if _, err := conn.Write(data[splitPt:]); err != nil {
		return fmt.Errorf("http trick part 2 write failed: %w", err)
	}

	return nil
}

func init() {
	RegisterStrategy(NewHTTPHostTrickStrategy(TrickHostCaseMix, 5))
}
