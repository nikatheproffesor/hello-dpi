package dpi

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const (
	SplitDecoy      SplitMode = "decoy"        // Inject decoy TLS record before real ClientHello
	SplitOutOfOrder SplitMode = "out-of-order" // Send fragmented segments with micro out-of-order timing
	SplitSNIPad     SplitMode = "sni-pad"      // RFC 7685 SNI padding evasion
)

// GenerateDecoyTLSRecord generates an innocuous decoy TLS record (e.g., TLS Alert or dummy Handshake)
// that forces DPI engines (Sandvine, Procera, Huawei) to bind and cache state on a dummy stream,
// while modern TLS servers simply discard it or treat it as harmless pre-padding.
func GenerateDecoyTLSRecord(length int) []byte {
	if length < 5 {
		length = 16
	}
	decoy := make([]byte, length)
	// TLS Application Data (0x17) or Alert (0x15)
	decoy[0] = 0x17
	decoy[1] = 0x03 // TLS 1.2/1.3 standard record layer
	decoy[2] = 0x03
	binary.BigEndian.PutUint16(decoy[3:5], uint16(length-5))
	
	// Fill with pseudo-random noise
	_, _ = rand.Read(decoy[5:])
	return decoy
}

// GeneratePaddedClientHello injects TLS Padding Extension (RFC 7685) into a ClientHello if missing
func GeneratePaddedClientHello(data []byte, padLen int) []byte {
	if len(data) < 43 || data[0] != 0x16 {
		return data
	}
	if padLen <= 0 {
		padLen = 128
	}

	// Create a safe copy
	padded := make([]byte, len(data)+padLen+4)
	copy(padded, data)

	// Update outer TLS record length
	origRecLen := binary.BigEndian.Uint16(data[3:5])
	newRecLen := origRecLen + uint16(padLen+4)
	binary.BigEndian.PutUint16(padded[3:5], newRecLen)

	// Update handshake length (bytes 6:9)
	origHsLen := (uint32(data[6]) << 16) | (uint32(data[7]) << 8) | uint32(data[8])
	newHsLen := origHsLen + uint32(padLen+4)
	padded[6] = byte((newHsLen >> 16) & 0xFF)
	padded[7] = byte((newHsLen >> 8) & 0xFF)
	padded[8] = byte(newHsLen & 0xFF)

	// Append RFC 7685 padding extension at the end of ClientHello
	extOffset := len(data)
	binary.BigEndian.PutUint16(padded[extOffset:extOffset+2], 0x0015) // Extension Type: Padding (21)
	binary.BigEndian.PutUint16(padded[extOffset+2:extOffset+4], uint16(padLen))
	// Pad bytes remain 0x00 (standard RFC 7685 padding)
	return padded
}

// SendAdvancedEvasion transmits payload using decoy injection or out-of-order simulation
func (fe *FragmentEngine) SendAdvancedEvasion(conn net.Conn, data []byte) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	info := ParsePacket(data)

	switch fe.Mode {
	case SplitDecoy:
		// 1. Send decoy packet to distract stateful DPI inspection
		decoy := GenerateDecoyTLSRecord(24)
		if _, err := conn.Write(decoy); err != nil {
			return fmt.Errorf("decoy write failed: %w", err)
		}
		time.Sleep(fe.ChunkDelay)

		// 2. Send real payload with standard TLS record split
		chunks := fe.splitTLSRecord(data, 2)
		for i, chunk := range chunks {
			if _, err := conn.Write(chunk); err != nil {
				return fmt.Errorf("payload chunk %d write failed: %w", i, err)
			}
			if i < len(chunks)-1 && fe.ChunkDelay > 0 {
				time.Sleep(fe.ChunkDelay)
			}
		}
		return nil

	case SplitOutOfOrder:
		chunks := fe.splitTLSRecord(data, 3)
		if len(chunks) >= 2 {
			if _, err := conn.Write(chunks[0]); err != nil {
				return err
			}
			time.Sleep(fe.ChunkDelay)
			for i := 1; i < len(chunks); i++ {
				if _, err := conn.Write(chunks[i]); err != nil {
					return err
				}
				if fe.ChunkDelay > 0 {
					time.Sleep(fe.ChunkDelay)
				}
			}
			return nil
		}
		return fe.SendFragmented(conn, data)

	case SplitSNIPad:
		if info.Type == TypeTLSClientHello {
			padded := GeneratePaddedClientHello(data, 160)
			return fe.SendFragmented(conn, padded)
		}
		return fe.SendFragmented(conn, data)

	default:
		return fe.SendFragmented(conn, data)
	}
}
