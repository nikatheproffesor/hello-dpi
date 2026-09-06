package dpi

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// TLSRecordSplitStrategy implements RFC 5246 section 6.2.1 and RFC 8446 section 5.1
// record layer splitting. It splits the initial TLS ClientHello into two valid TLS records.
type TLSRecordSplitStrategy struct {
	splitOffset int
	delay       time.Duration
}

// NewTLSRecordSplitStrategy creates a TLS record splitting strategy.
func NewTLSRecordSplitStrategy(splitOffset int, delayMs int) *TLSRecordSplitStrategy {
	if splitOffset <= 0 {
		splitOffset = 5
	}
	if delayMs <= 0 {
		delayMs = 5
	}
	return &TLSRecordSplitStrategy{
		splitOffset: splitOffset,
		delay:       time.Duration(delayMs) * time.Millisecond,
	}
}

func (s *TLSRecordSplitStrategy) Name() string {
	return string(SplitTLS)
}

func (s *TLSRecordSplitStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	chunks := s.splitRecords(data, s.splitOffset)
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		if _, err := conn.Write(chunk); err != nil {
			return fmt.Errorf("tls record split chunk %d write failed: %w", i, err)
		}
		if i < len(chunks)-1 && s.delay > 0 {
			time.Sleep(s.delay)
		}
	}
	return nil
}

func (s *TLSRecordSplitStrategy) splitRecords(data []byte, splitPos int) [][]byte {
	if len(data) < 9 || data[0] != 0x16 {
		if len(data) <= 1 {
			return [][]byte{data}
		}
		return [][]byte{data[:1], data[1:]}
	}

	recLen := int(binary.BigEndian.Uint16(data[3:5]))
	if recLen <= 0 || len(data) < 6 {
		if len(data) <= 1 {
			return [][]byte{data}
		}
		return [][]byte{data[:1], data[1:]}
	}

	availablePayload := len(data) - 5
	if recLen > availablePayload {
		recLen = availablePayload
	}

	if splitPos <= 0 || splitPos >= recLen {
		splitPos = 1
	}

	// Record 1
	rec1 := make([]byte, 5+splitPos)
	rec1[0] = data[0]
	rec1[1] = data[1]
	rec1[2] = data[2]
	binary.BigEndian.PutUint16(rec1[3:5], uint16(splitPos))
	copy(rec1[5:], data[5:5+splitPos])

	// Record 2
	remLen := recLen - splitPos
	rec2 := make([]byte, 5+remLen)
	rec2[0] = data[0]
	rec2[1] = data[1]
	rec2[2] = data[2]
	binary.BigEndian.PutUint16(rec2[3:5], uint16(remLen))
	copy(rec2[5:], data[5+splitPos:5+recLen])

	extraLen := len(data) - (5 + recLen)
	if extraLen > 0 {
		recExtra := make([]byte, extraLen)
		copy(recExtra, data[5+recLen:])
		return [][]byte{rec1, rec2, recExtra}
	}

	return [][]byte{rec1, rec2}
}

func init() {
	RegisterStrategy(NewTLSRecordSplitStrategy(5, 5))
}
