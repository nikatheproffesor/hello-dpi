package dpi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// SplitMode specifies how packets are split
type SplitMode string

const (
	SplitAuto      SplitMode = "auto"       // Automatic adaptive fragmentation
	SplitTLS       SplitMode = "tlsrec"     // Split TLS ClientHello into two valid TLS records
	SplitSNI       SplitMode = "sni"        // Split at the SNI hostname
	SplitFirstByte SplitMode = "first-byte" // Split 1st byte from the rest
	SplitChunked   SplitMode = "chunked"    // Split into small fragments (e.g. 20-50 bytes)
	SplitHTTPMix   SplitMode = "http-mix"   // Apply HTTP Host header tricks
)

// FragmentEngine handles sending data across the TCP socket in fragments to evade DPI
type FragmentEngine struct {
	Mode         SplitMode
	ChunkDelay   time.Duration // Delay between chunks to guarantee separate TCP segments
	CustomOffset int           // Manual split offset (0 means auto)
}

// NewFragmentEngine creates a configured FragmentEngine
func NewFragmentEngine(mode SplitMode, delayMs int) *FragmentEngine {
	if delayMs <= 0 {
		delayMs = 5 // 5ms default is reliable across all OS TCP stacks
	}
	if mode == "" {
		mode = SplitAuto
	}
	return &FragmentEngine{
		Mode:       mode,
		ChunkDelay: time.Duration(delayMs) * time.Millisecond,
	}
}

// SendFragmented sends the initial payload to the target server in fragmented form
func (fe *FragmentEngine) SendFragmented(conn net.Conn, data []byte) error {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		// Disable Nagle's algorithm so chunks are sent immediately in individual packets
		_ = tcpConn.SetNoDelay(true)
	}

	info := ParsePacket(data)

	var chunks [][]byte

	switch info.Type {
	case TypeTLSClientHello:
		chunks = fe.fragmentTLS(data, info)
	case TypeHTTPRequest:
		chunks = fe.fragmentHTTP(data, info)
	default:
		chunks = fe.splitFirstByte(data)
	}

	// Transmit each chunk over the connection
	for i, chunk := range chunks {
		if len(chunk) == 0 {
			continue
		}
		_, err := conn.Write(chunk)
		if err != nil {
			return fmt.Errorf("failed writing fragment %d: %w", i, err)
		}

		// Wait briefly between chunks to prevent the OS TCP stack from merging them
		if i < len(chunks)-1 && fe.ChunkDelay > 0 {
			time.Sleep(fe.ChunkDelay)
		}
	}

	return nil
}

// fragmentTLS creates fragmented chunks for TLS ClientHello
func (fe *FragmentEngine) fragmentTLS(data []byte, info ParsedInfo) [][]byte {
	splitPos := 5
	if fe.CustomOffset > 0 {
		splitPos = fe.CustomOffset
	}

	switch fe.Mode {
	case SplitAuto, SplitTLS, SplitSNI:
		return fe.splitTLSRecord(data, splitPos)
	case SplitChunked:
		return fe.splitInChunks(data, 40)
	case SplitFirstByte:
		return fe.splitFirstByte(data)
	default:
		return fe.splitTLSRecord(data, splitPos)
	}
}

// splitTLSRecord splits a single TLS record into two valid RFC-compliant TLS records.
// Compliant with RFC 5246 section 6.2.1 and RFC 8446 section 5.1.
// Panic-proof and handles incomplete or malformed inputs safely.
func (fe *FragmentEngine) splitTLSRecord(data []byte, splitPos int) [][]byte {
	if len(data) < 9 || data[0] != 0x16 {
		return fe.splitFirstByte(data)
	}

	recLen := int(binary.BigEndian.Uint16(data[3:5]))
	if recLen <= 0 || len(data) < 6 {
		return fe.splitFirstByte(data)
	}

	availablePayload := len(data) - 5
	if recLen > availablePayload {
		recLen = availablePayload
	}

	if splitPos <= 0 || splitPos >= recLen {
		splitPos = 1
	}

	// Record 1: 5-byte header + first splitPos bytes
	rec1 := make([]byte, 5+splitPos)
	rec1[0] = data[0] // 0x16 (Handshake)
	rec1[1] = data[1] // TLS major
	rec1[2] = data[2] // TLS minor
	binary.BigEndian.PutUint16(rec1[3:5], uint16(splitPos))
	copy(rec1[5:], data[5:5+splitPos])

	// Record 2: 5-byte header + remaining handshake bytes
	remRecLen := recLen - splitPos
	extraLen := len(data) - (5 + recLen)
	rec2 := make([]byte, 5+remRecLen+extraLen)
	rec2[0] = data[0]
	rec2[1] = data[1]
	rec2[2] = data[2]
	binary.BigEndian.PutUint16(rec2[3:5], uint16(remRecLen))
	copy(rec2[5:5+remRecLen], data[5+splitPos:5+recLen])
	if extraLen > 0 {
		copy(rec2[5+remRecLen:], data[5+recLen:])
	}

	return [][]byte{rec1, rec2}
}

// fragmentHTTP modifies or fragments HTTP requests
func (fe *FragmentEngine) fragmentHTTP(data []byte, info ParsedInfo) [][]byte {
	if info.HostOffset >= 0 && info.HostOffset+5 < len(data) {
		modData := make([]byte, len(data))
		copy(modData, data)

		if bytes.Equal(bytes.ToLower(modData[info.HostOffset:info.HostOffset+5]), []byte("host:")) {
			modData[info.HostOffset] = 'h'
			modData[info.HostOffset+1] = 'o'
			modData[info.HostOffset+2] = 's'
			modData[info.HostOffset+3] = 't'
		}

		splitPoint := info.HostOffset + 3
		if splitPoint > 0 && splitPoint < len(modData) {
			return [][]byte{
				modData[:splitPoint],
				modData[splitPoint:],
			}
		}
	}

	return fe.splitFirstByte(data)
}

// splitFirstByte splits the 1st byte and the rest
func (fe *FragmentEngine) splitFirstByte(data []byte) [][]byte {
	if len(data) <= 1 {
		return [][]byte{data}
	}
	return [][]byte{
		data[:1],
		data[1:],
	}
}

// splitInChunks splits data into fixed size chunks
func (fe *FragmentEngine) splitInChunks(data []byte, chunkSize int) [][]byte {
	if chunkSize <= 0 || len(data) <= chunkSize {
		return [][]byte{data}
	}
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}
