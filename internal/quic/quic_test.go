package quic

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

// helper to build a valid synthetic QUIC Initial packet
func buildSyntheticQUICInitial(dcid, scid []byte) []byte {
	dcidLen := len(dcid)
	scidLen := len(scid)
	pkt := make([]byte, 1+4+1+dcidLen+1+scidLen+1+2+100)
	pkt[0] = 0xc0 // Long header | Initial
	binary.BigEndian.PutUint32(pkt[1:5], Version1)
	pkt[5] = byte(dcidLen)
	copy(pkt[6:6+dcidLen], dcid)
	offset := 6 + dcidLen
	pkt[offset] = byte(scidLen)
	offset++
	copy(pkt[offset:offset+scidLen], scid)
	offset += scidLen
	pkt[offset] = 0x00 // Token length
	offset++
	binary.BigEndian.PutUint16(pkt[offset:offset+2], 0x4064) // Payload len 100
	return pkt
}

func TestQUICInitialDetection(t *testing.T) {
	dcid := []byte{0x01, 0x02, 0x03, 0x04}
	scid := []byte{0x05, 0x06, 0x07, 0x08}
	raw := buildSyntheticQUICInitial(dcid, scid)

	if !IsQUIC(raw) {
		t.Fatalf("expected IsQUIC to return true")
	}
	if !IsQUICInitial(raw) {
		t.Fatalf("expected IsQUICInitial to return true")
	}

	hdr, err := ParseInitial(raw)
	if err != nil {
		t.Fatalf("ParseInitial failed: %v", err)
	}
	if hdr.Version != Version1 {
		t.Errorf("expected version %x, got %x", Version1, hdr.Version)
	}
	if !bytes.Equal(hdr.DCID, dcid) {
		t.Errorf("expected DCID %x, got %x", dcid, hdr.DCID)
	}
	if !bytes.Equal(hdr.SCID, scid) {
		t.Errorf("expected SCID %x, got %x", scid, hdr.SCID)
	}
}

func TestDecoyInitial(t *testing.T) {
	dcid := []byte{0xaa, 0xbb, 0xcc, 0xdd}
	decoy := BuildDecoyInitial(dcid)

	if !IsQUICInitial(decoy) {
		t.Fatalf("expected decoy to be recognized as QUIC initial packet")
	}

	hdr, err := ParseInitial(decoy)
	if err != nil {
		t.Fatalf("failed to parse decoy: %v", err)
	}
	if hdr.Version != VersionGrease {
		t.Errorf("expected GREASE version %x, got %x", VersionGrease, hdr.Version)
	}
	if !bytes.Equal(hdr.DCID, dcid) {
		t.Errorf("expected DCID in decoy to match original: %x vs %x", dcid, hdr.DCID)
	}
}

func TestMangler(t *testing.T) {
	mangler := NewMangler()

	dcid := []byte{0x11, 0x22, 0x33, 0x44}
	initialPkt := buildSyntheticQUICInitial(dcid, []byte{0x99})
	targetAddr := &net.UDPAddr{IP: net.ParseIP("1.1.1.1"), Port: 443}

	packets := mangler.ProcessOutbound(targetAddr, initialPkt)
	if len(packets) != 2 {
		t.Fatalf("expected 2 packets (decoy + real), got %d", len(packets))
	}

	// First packet is decoy
	if binary.BigEndian.Uint32(packets[0][1:5]) != VersionGrease {
		t.Errorf("first packet must be GREASE decoy")
	}
	// Second packet is real
	if !bytes.Equal(packets[1], initialPkt) {
		t.Errorf("second packet must match real initial packet")
	}

	// Discord voice port test (should pass through without decoy)
	voiceAddr := &net.UDPAddr{IP: net.ParseIP("162.159.135.232"), Port: 50005}
	voiceData := []byte("discord-voice-rtp-data")
	out := mangler.ProcessOutbound(voiceAddr, voiceData)
	if len(out) != 1 || !bytes.Equal(out[0], voiceData) {
		t.Errorf("voice data must pass through unmodified")
	}
}
