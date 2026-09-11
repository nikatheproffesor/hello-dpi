package quic

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

// Common QUIC Versions (RFC 9000, RFC 9369)
const (
	Version1       uint32 = 0x00000001
	Version2       uint32 = 0x709a50c4
	VersionDraft29 uint32 = 0xff00001d
	VersionGrease  uint32 = 0x0a0a0a0a
)

var (
	ErrNotQUICLongHeader = errors.New("packet is not a QUIC long header")
	ErrInvalidPacketLen  = errors.New("packet too short for QUIC header")
	ErrNotInitial        = errors.New("packet is not a QUIC Initial packet")
)

// Header represents parsed components of a QUIC Long Header
type Header struct {
	IsLongHeader bool
	Type         byte // 0x00: Initial, 0x01: 0-RTT, 0x02: Handshake, 0x03: Retry
	Version      uint32
	DCID         []byte
	SCID         []byte
	Token        []byte
	Payload      []byte
}

// IsQUIC checks if the UDP payload looks like a QUIC packet
func IsQUIC(payload []byte) bool {
	if len(payload) < 5 {
		return false
	}
	// Bit 7 must be 1 for Long Header, Bit 6 (Fixed bit) must be 1
	firstByte := payload[0]
	if (firstByte & 0x80) != 0 {
		return (firstByte & 0x40) != 0
	}
	// Short header (1-RTT) also has Fixed Bit (bit 6) set
	return (firstByte & 0x40) != 0
}

// IsQUICInitial checks specifically for a QUIC Initial packet
func IsQUICInitial(payload []byte) bool {
	if len(payload) < 7 {
		return false
	}
	firstByte := payload[0]
	// Must be Long Header (0x80), Fixed Bit set (0x40), and Type == 0x00 (bits 4-5 are 0)
	if (firstByte&0x80) == 0 || (firstByte&0x40) == 0 {
		return false
	}
	packetType := (firstByte & 0x30) >> 4
	if packetType != 0x00 {
		return false
	}
	version := binary.BigEndian.Uint32(payload[1:5])
	return version != 0 // Version 0 is Version Negotiation
}

// ParseInitial extracts DCID and SCID from a QUIC Initial packet
func ParseInitial(payload []byte) (*Header, error) {
	if len(payload) < 7 {
		return nil, ErrInvalidPacketLen
	}
	firstByte := payload[0]
	if (firstByte & 0x80) == 0 {
		return nil, ErrNotQUICLongHeader
	}
	packetType := (firstByte & 0x30) >> 4
	if packetType != 0x00 {
		return nil, ErrNotInitial
	}

	version := binary.BigEndian.Uint32(payload[1:5])
	dcidLen := int(payload[5])
	offset := 6
	if len(payload) < offset+dcidLen+1 {
		return nil, ErrInvalidPacketLen
	}
	dcid := payload[offset : offset+dcidLen]
	offset += dcidLen

	scidLen := int(payload[offset])
	offset++
	if len(payload) < offset+scidLen {
		return nil, ErrInvalidPacketLen
	}
	scid := payload[offset : offset+scidLen]
	offset += scidLen

	return &Header{
		IsLongHeader: true,
		Type:         0x00,
		Version:      version,
		DCID:         dcid,
		SCID:         scid,
		Payload:      payload[offset:],
	}, nil
}

// BuildDecoyInitial constructs a GREASE (RFC 8701) decoy QUIC Initial packet
// that confuses Middlebox / DPI state machines when sent immediately prior to the real packet.
func BuildDecoyInitial(dcid []byte) []byte {
	// Header: [Flags 1B][Version 4B][DCIDLen 1B][DCID NB][SCIDLen 1B][SCID 8B][TokenLen 1B][Length 2B][Payload...]
	dcidLen := len(dcid)
	if dcidLen > 20 {
		dcidLen = 20
	}
	scidLen := 8

	decoy := make([]byte, 1+4+1+dcidLen+1+scidLen+1+2+64)
	decoy[0] = 0xc0 // Long header (0x80) | Fixed Bit (0x40) | Initial (0x00)
	binary.BigEndian.PutUint32(decoy[1:5], VersionGrease)
	decoy[5] = byte(dcidLen)
	if dcidLen > 0 {
		copy(decoy[6:6+dcidLen], dcid[:dcidLen])
	}
	offset := 6 + dcidLen
	decoy[offset] = byte(scidLen)
	offset++
	_, _ = rand.Read(decoy[offset : offset+scidLen])
	offset += scidLen

	decoy[offset] = 0x00 // Token Length = 0
	offset++
	binary.BigEndian.PutUint16(decoy[offset:offset+2], 0x4040) // Varint length ~64
	offset += 2
	_, _ = rand.Read(decoy[offset:])

	return decoy
}

// MangleQUICInitial produces an evasive packet sequence for QUIC Initial handshakes.
// It returns a slice of UDP datagrams to send: [Decoy, RealPayload].
func MangleQUICInitial(payload []byte) [][]byte {
	hdr, err := ParseInitial(payload)
	if err != nil {
		// Not a parsable Initial, return as-is
		return [][]byte{payload}
	}

	decoy := BuildDecoyInitial(hdr.DCID)
	return [][]byte{decoy, payload}
}

// IsDiscordVoicePort checks if the target port belongs to Discord or game voice (WebRTC/RTC/UDP)
func IsDiscordVoicePort(port int) bool {
	// Discord RTC media server ports typically range from 50001 to 65535 or STUN 3478
	return port == 3478 || (port >= 50000 && port <= 65535)
}

// IsQUICPort checks if target port is typical for QUIC / HTTP-3 (443, 8443)
func IsQUICPort(port int) bool {
	return port == 443 || port == 8443
}

// Mangler coordinates UDP packet evasion strategies
type Mangler struct {
	mu           sync.RWMutex
	enableDecoy  bool
	decoyDelayMs int
}

// NewMangler creates a new QUIC / UDP packet mangler
func NewMangler() *Mangler {
	return &Mangler{
		enableDecoy:  true,
		decoyDelayMs: 2,
	}
}

// ProcessOutbound handles outbound UDP packets from client.
// Returns one or more UDP datagrams to transmit to the destination.
func (m *Mangler) ProcessOutbound(destAddr *net.UDPAddr, payload []byte) [][]byte {
	if destAddr == nil || len(payload) == 0 {
		return [][]byte{payload}
	}

	// 1. Check for QUIC Initial handshake packet
	if IsQUICPort(destAddr.Port) && IsQUICInitial(payload) {
		m.mu.RLock()
		decoyOn := m.enableDecoy
		m.mu.RUnlock()

		if decoyOn {
			return MangleQUICInitial(payload)
		}
	}

	// 2. Regular UDP / Voice / Discord WebRTC: Pass through with 0 ms latency
	return [][]byte{payload}
}

// TransmitOutbound sends the processed datagrams to the destination UDP socket.
func (m *Mangler) TransmitOutbound(conn *net.UDPConn, destAddr *net.UDPAddr, packets [][]byte) error {
	for i, pkt := range packets {
		if i > 0 && m.decoyDelayMs > 0 {
			time.Sleep(time.Duration(m.decoyDelayMs) * time.Millisecond)
		}
		if _, err := conn.WriteToUDP(pkt, destAddr); err != nil {
			return err
		}
	}
	return nil
}
