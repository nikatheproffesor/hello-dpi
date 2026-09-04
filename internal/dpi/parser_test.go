package dpi

import (
	"testing"
)

func TestExtractHTTPHost(t *testing.T) {
	req := []byte("GET / HTTP/1.1\r\nHost: example.com\r\nUser-Agent: curl/7.68.0\r\nAccept: */*\r\n\r\n")
	info := ParsePacket(req)

	if info.Type != TypeHTTPRequest {
		t.Fatalf("Expected TypeHTTPRequest, got %v", info.Type)
	}
	if info.Host != "example.com" {
		t.Fatalf("Expected host example.com, got '%s'", info.Host)
	}
}

func TestSplitFirstByte(t *testing.T) {
	fe := NewFragmentEngine(SplitFirstByte, 0)
	data := []byte("HELLO WORLD")
	chunks := fe.splitFirstByte(data)

	if len(chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(chunks))
	}
	if string(chunks[0]) != "H" {
		t.Fatalf("Expected 'H', got '%s'", string(chunks[0]))
	}
	if string(chunks[1]) != "ELLO WORLD" {
		t.Fatalf("Expected 'ELLO WORLD', got '%s'", string(chunks[1]))
	}
}
