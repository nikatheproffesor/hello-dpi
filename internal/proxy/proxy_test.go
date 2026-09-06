package proxy

import (
	"bufio"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/hellodpi/hellodpi/internal/dpi"
)

func TestProxyLoopDetection(t *testing.T) {
	srv := NewServer(Config{Addr: "127.0.0.1:8080"})

	if !srv.isProxyLoop("127.0.0.1", "8080") {
		t.Errorf("Expected loop to be detected for 127.0.0.1:8080")
	}
	if !srv.isProxyLoop("localhost", "8080") {
		t.Errorf("Expected loop to be detected for localhost:8080")
	}
	if srv.isProxyLoop("discord.com", "443") {
		t.Errorf("External target should NOT be detected as loop")
	}
}

func TestSOCKS5HandshakeNegotiation(t *testing.T) {
	srv := NewServer(Config{
		Addr:      "127.0.0.1:0",
		SplitMode: dpi.SplitTLS,
		DelayMs:   5,
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	srv.listener = l
	srv.Addr = l.Addr().String()

	go func() {
		conn, err := l.Accept()
		if err == nil {
			reader := bufio.NewReader(conn)
			srv.handleSOCKS5(conn, reader)
		}
	}()

	client, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer client.Close()

	// SOCKS5 greeting: 1 method (0x00 No Auth)
	_, _ = client.Write([]byte{0x05, 0x01, 0x00})

	resp := make([]byte, 2)
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))
	n, err := client.Read(resp)
	if err != nil || n != 2 {
		t.Fatalf("Failed to read SOCKS5 greeting response: %v", err)
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		t.Fatalf("Unexpected greeting response: %x %x", resp[0], resp[1])
	}
}

func TestHTTPControlRouting(t *testing.T) {
	srv := NewServer(Config{
		Addr:      "127.0.0.1:0",
		SplitMode: dpi.SplitAuto,
	})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	srv.listener = l
	srv.Addr = l.Addr().String()

	go func() {
		conn, err := l.Accept()
		if err == nil {
			reader := bufio.NewReader(conn)
			srv.handleHTTP(conn, reader)
		}
	}()

	client, err := net.Dial("tcp", srv.Addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer client.Close()

	// Send request to /api/status
	_, _ = client.Write([]byte("GET /api/status HTTP/1.1\r\nHost: 127.0.0.1\r\n\r\n"))

	reader := bufio.NewReader(client)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("Failed to read HTTP response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for /api/status, got %d", resp.StatusCode)
	}
}
