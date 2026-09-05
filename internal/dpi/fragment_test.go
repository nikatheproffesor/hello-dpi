package dpi

import (
	"encoding/binary"
	"testing"
)

func TestSplitTLSRecordValid(t *testing.T) {
	fe := NewFragmentEngine(SplitTLS, 5)

	// Create valid mock TLS ClientHello record
	payload := []byte{0x01, 0x00, 0x00, 0x20, 0x03, 0x03, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	recLen := uint16(len(payload))

	data := make([]byte, 5+len(payload))
	data[0] = 0x16
	data[1] = 0x03
	data[2] = 0x01
	binary.BigEndian.PutUint16(data[3:5], recLen)
	copy(data[5:], payload)

	chunks := fe.splitTLSRecord(data, 5)
	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(chunks))
	}

	// Chunk 1 must be a valid TLS record with length 5
	if chunks[0][0] != 0x16 {
		t.Fatalf("Chunk 1 not TLS record: %x", chunks[0][0])
	}
	len1 := binary.BigEndian.Uint16(chunks[0][3:5])
	if len1 != 5 {
		t.Fatalf("Chunk 1 expected length 5, got %d", len1)
	}

	// Chunk 2 must be a valid TLS record with remaining length
	if chunks[1][0] != 0x16 {
		t.Fatalf("Chunk 2 not TLS record: %x", chunks[1][0])
	}
	len2 := binary.BigEndian.Uint16(chunks[1][3:5])
	if len2 != recLen-5 {
		t.Fatalf("Chunk 2 expected length %d, got %d", recLen-5, len2)
	}

	// Reassembled payload must match original
	reassembled := append(chunks[0][5:], chunks[1][5:]...)
	if string(reassembled) != string(payload) {
		t.Fatalf("Payload mismatch after reassembly")
	}
}

func TestSplitTLSRecordTruncated(t *testing.T) {
	fe := NewFragmentEngine(SplitTLS, 5)

	// Incomplete record should fallback gracefully to splitFirstByte without panic
	data := []byte{0x16, 0x03, 0x01, 0x00, 0x20, 0x01}
	chunks := fe.splitTLSRecord(data, 5)
	if len(chunks) == 0 {
		t.Fatalf("Expected at least 1 chunk, got 0")
	}
}
