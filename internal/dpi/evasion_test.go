package dpi

import (
	"net"
	"testing"
	"time"
)

func TestGenerateDecoyTLSRecord(t *testing.T) {
	decoy := GenerateDecoyTLSRecord(20)
	if len(decoy) != 20 {
		t.Fatalf("expected decoy length 20, got %d", len(decoy))
	}
	if decoy[0] != 0x17 {
		t.Fatalf("expected TLS Application data header 0x17, got %x", decoy[0])
	}
}

func TestGeneratePaddedClientHello(t *testing.T) {
	// Dummy minimal TLS ClientHello (type 0x16, version 0x0303, len 40)
	raw := make([]byte, 50)
	raw[0] = 0x16
	raw[1] = 0x03
	raw[2] = 0x03
	raw[5] = 0x01 // Handshake ClientHello

	padded := GeneratePaddedClientHello(raw, 64)
	if len(padded) <= len(raw) {
		t.Fatalf("expected padded length > %d, got %d", len(raw), len(padded))
	}
}

func TestSendAdvancedEvasion(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	fe := NewFragmentEngine(SplitDecoy, 1)

	payload := make([]byte, 60)
	payload[0] = 0x16
	payload[1] = 0x03
	payload[2] = 0x03
	payload[3] = 0x00
	payload[4] = 55 // record length

	go func() {
		_ = fe.SendAdvancedEvasion(client, payload)
	}()

	buf := make([]byte, 200)
	_ = server.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := server.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("failed reading from mock pipe: %v", err)
	}

	// Decoy should be received first (starts with 0x17)
	if buf[0] != 0x17 {
		t.Fatalf("expected decoy record 0x17, got 0x%x", buf[0])
	}
}
