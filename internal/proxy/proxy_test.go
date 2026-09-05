package proxy

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

func TestDirectPassThrough(t *testing.T) {
	cases := []struct {
		host     string
		expected bool
	}{
		{"wifi.gsb.gov.tr", true},
		{"portal.kyk.gov.tr", true},
		{"captive.apple.com", true},
		{"connectivitycheck.gstatic.com", true},
		{"msftconnecttest.com", true},
		{"10.10.1.1", true},
		{"192.168.1.1", true},
		{"172.20.0.1", true},
		{"localhost", true},
		{"mycomputer.local", true},
		{"discord.com", false},
		{"w2g.tv", false},
		{"cloudflare.com", false},
		{"1.1.1.1", false},
	}

	for _, tc := range cases {
		actual := isDirectPassThrough(tc.host)
		if actual != tc.expected {
			t.Errorf("isDirectPassThrough(%q) = %v, expected %v", tc.host, actual, tc.expected)
		}
	}
}

// dummyConn implements net.Conn over in-memory buffers
type dummyConn struct {
	net.Conn
	r io.Reader
	w *bytes.Buffer
}

func (d *dummyConn) Read(b []byte) (int, error) {
	return d.r.Read(b)
}

func (d *dummyConn) Write(b []byte) (int, error) {
	return d.w.Write(b)
}

func (d *dummyConn) Close() error {
	return nil
}

func (d *dummyConn) SetDeadline(t time.Time) error {
	return nil
}

func TestBufferedConnPreservesPipelinedBytes(t *testing.T) {
	// Simulate client sending 20 bytes, but reader buffers 15 bytes in Peek
	data := []byte("1234567890ABCDEFGHIJ")
	rawConn := &dummyConn{
		r: bytes.NewReader(data),
		w: new(bytes.Buffer),
	}

	reader := bufio.NewReader(rawConn)
	// Read first 5 bytes via reader
	peeked, err := reader.Peek(5)
	if err != nil || string(peeked) != "12345" {
		t.Fatalf("peek failed: %v", err)
	}

	// Consume first 5 bytes
	first5 := make([]byte, 5)
	if _, err := io.ReadFull(reader, first5); err != nil {
		t.Fatalf("read first 5 failed: %v", err)
	}

	// Now wrap with bufferedConn
	bc := &bufferedConn{r: reader, Conn: rawConn}

	// The remaining 15 bytes should be read without ANY data loss!
	remaining := make([]byte, 15)
	n, err := io.ReadFull(bc, remaining)
	if err != nil || n != 15 {
		t.Fatalf("expected 15 bytes, got %d (err: %v)", n, err)
	}

	if string(remaining) != "67890ABCDEFGHIJ" {
		t.Fatalf("expected '67890ABCDEFGHIJ', got '%s'", string(remaining))
	}
}
