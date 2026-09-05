package dpi

import (
	"bytes"
	"encoding/binary"
	"strings"
)

// PacketType identifies the detected protocol of a payload.
type PacketType int

const (
	TypeUnknown PacketType = iota
	TypeTLSClientHello
	TypeHTTPRequest
)

// ParsedInfo contains information extracted from the initial packet.
type ParsedInfo struct {
	Type       PacketType
	Host       string
	SNIOffset  int // Start index of the SNI hostname in the payload
	SNILength  int // Length of the SNI hostname in the payload
	HostOffset int // Start index of "Host: " in HTTP payload
}

// ParsePacket inspects the initial bytes sent after connection establishment.
func ParsePacket(data []byte) ParsedInfo {
	if len(data) < 5 {
		return ParsedInfo{Type: TypeUnknown}
	}

	// 1. Check for TLS Handshake (0x16) and ClientHello (0x01)
	if data[0] == 0x16 && len(data) >= 9 && data[5] == 0x01 {
		info := ParsedInfo{Type: TypeTLSClientHello}
		sni, offset, length := extractSNI(data)
		if sni != "" {
			info.Host = sni
			info.SNIOffset = offset
			info.SNILength = length
		}
		return info
	}

	// 2. Check for HTTP Methods
	methods := []string{"GET ", "POST ", "HEAD ", "OPTIONS ", "PUT ", "DELETE ", "CONNECT "}
	for _, m := range methods {
		if strings.HasPrefix(string(data[:min(len(data), 10)]), m) {
			info := ParsedInfo{Type: TypeHTTPRequest}
			host, offset := extractHTTPHost(data)
			info.Host = host
			info.HostOffset = offset
			return info
		}
	}

	return ParsedInfo{Type: TypeUnknown}
}

// extractSNI parses the TLS ClientHello record to find the Server Name Indication (SNI)
func extractSNI(data []byte) (string, int, int) {
	if len(data) < 43 {
		return "", 0, 0
	}

	// Skip TLS Record Header (5 bytes): Type (1) + Version (2) + Length (2)
	// Skip Handshake Header (4 bytes): Type (1) + Length (3)
	// Skip Client Version (2) + Random (32) = 34 bytes
	pos := 5 + 4 + 2 + 32
	if pos >= len(data) {
		return "", 0, 0
	}

	// Session ID Length
	sessionIDLen := int(data[pos])
	pos += 1 + sessionIDLen
	if pos+2 > len(data) {
		return "", 0, 0
	}

	// Cipher Suites Length
	cipherSuitesLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2 + cipherSuitesLen
	if pos+1 > len(data) {
		return "", 0, 0
	}

	// Compression Methods Length
	compressionLen := int(data[pos])
	pos += 1 + compressionLen
	if pos+2 > len(data) {
		return "", 0, 0
	}

	// Extensions Length
	extensionsLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
	pos += 2
	end := pos + extensionsLen
	if end > len(data) {
		end = len(data)
	}

	// Walk Extensions
	for pos+4 <= end {
		extType := binary.BigEndian.Uint16(data[pos : pos+2])
		extLen := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		pos += 4

		if pos+extLen > end {
			break
		}

		// Extension 0x0000 is Server Name Indication (SNI)
		if extType == 0 {
			sniData := data[pos : pos+extLen]
			if len(sniData) >= 5 {
				// server_name_list length (2) + name_type (1, 0=host_name) + name_length (2)
				nameType := sniData[2]
				nameLen := int(binary.BigEndian.Uint16(sniData[3:5]))
				if nameType == 0 && len(sniData) >= 5+nameLen {
					sniOffset := pos + 5
					hostname := string(sniData[5 : 5+nameLen])
					return hostname, sniOffset, nameLen
				}
			}
		}
		pos += extLen
	}

	return "", 0, 0
}

// extractHTTPHost finds the Host header in plain HTTP requests
func extractHTTPHost(data []byte) (string, int) {
	lower := bytes.ToLower(data)
	idx := bytes.Index(lower, []byte("\r\nhost:"))
	if idx == -1 {
		idx = bytes.Index(lower, []byte("\nhost:"))
		if idx == -1 {
			return "", -1
		}
		idx += 1
	} else {
		idx += 2
	}

	// idx points to "host:"
	lineEnd := bytes.Index(data[idx:], []byte("\n"))
	if lineEnd == -1 {
		lineEnd = len(data) - idx
	}
	line := string(data[idx : idx+lineEnd])
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1]), idx
	}
	return "", idx
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
