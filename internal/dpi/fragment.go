package dpi

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

// SplitMode specifies how packets are split
type SplitMode string

const (
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
		delayMs = 2 // 2ms is optimal: fast enough to be unnoticeable, slow enough to ensure separate TCP packets
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
		// Fallback for unknown protocols: split first byte
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
	switch fe.Mode {
	case SplitSNI:
		if info.SNIOffset > 0 && info.SNILength > 1 {
			// Split right in the middle of the SNI domain name
			// e.g. "wiki" in first packet, "pedia.org" in second packet
			splitPoint := info.SNIOffset + (info.SNILength / 2)
			if splitPoint > 0 && splitPoint < len(data) {
				return [][]byte{
					data[:splitPoint],
					data[splitPoint:],
				}
			}
		}
		// If SNI couldn't be located precisely, fall back to first byte split
		return fe.splitFirstByte(data)

	case SplitChunked:
		return fe.splitInChunks(data, 40)

	case SplitFirstByte:
		fallthrough
	default:
		return fe.splitFirstByte(data)
	}
}

// fragmentHTTP modifies or fragments HTTP requests
func (fe *FragmentEngine) fragmentHTTP(data []byte, info ParsedInfo) [][]byte {
	// If Host offset is found, we can trick DPI by altering Host case or adding spaces
	if info.HostOffset >= 0 && info.HostOffset+5 < len(data) {
		modData := make([]byte, len(data))
		copy(modData, data)

		// Replace "Host:" with "host:" (standard HTTP servers accept this, many DPI boxes don't match)
		if bytes.Equal(bytes.ToLower(modData[info.HostOffset:info.HostOffset+5]), []byte("host:")) {
			modData[info.HostOffset] = 'h'
			modData[info.HostOffset+1] = 'o'
			modData[info.HostOffset+2] = 's'
			modData[info.HostOffset+3] = 't'
		}

		// Split right inside the host header
		splitPoint := info.HostOffset + 3
		return [][]byte{
			modData[:splitPoint],
			modData[splitPoint:],
		}
	}

	return fe.splitFirstByte(data)
}

// splitFirstByte splits the 1st byte (or 2 bytes) and the rest
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
