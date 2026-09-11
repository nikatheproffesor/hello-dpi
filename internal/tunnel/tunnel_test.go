package tunnel

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestBufferedConn(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	var serverConn net.Conn
	go func() {
		defer close(done)
		var err error
		serverConn, err = ln.Accept()
		if err != nil {
			return
		}
		defer serverConn.Close()
		_, _ = serverConn.Write([]byte("WORLD"))
		if tc, ok := serverConn.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer clientConn.Close()

	// Put "HELLO " in a pre-buffer, and serverConn streams "WORLD"
	preBuf := bytes.NewReader([]byte("HELLO "))
	reader := bufio.NewReader(io.MultiReader(preBuf, clientConn))

	bc := NewBufferedConn(reader, clientConn)

	data := make([]byte, 11)
	n, err := io.ReadFull(bc, data)
	if err != nil {
		t.Fatalf("ReadFull failed (n=%d, read=%q): %v", n, string(data[:n]), err)
	}
	if n != 11 || string(data) != "HELLO WORLD" {
		t.Fatalf("Expected 'HELLO WORLD', got '%s'", string(data))
	}
	<-done
}

func TestReadInitialPayloadTLS(t *testing.T) {
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	rec := make([]byte, 5+len(payload))
	rec[0] = 0x16
	rec[1] = 0x03
	rec[2] = 0x03
	binary.BigEndian.PutUint16(rec[3:5], uint16(len(payload)))
	copy(rec[5:], payload)

	reader := bufio.NewReader(bytes.NewReader(rec))
	extracted, err := ReadInitialPayload(reader)
	if err != nil {
		t.Fatalf("ReadInitialPayload failed: %v", err)
	}
	if !bytes.Equal(extracted, rec) {
		t.Fatalf("Extracted payload mismatch")
	}
}

func TestPipeBidirectional(t *testing.T) {
	c1a, c1b := net.Pipe()
	c2a, c2b := net.Pipe()

	defer c1a.Close()
	defer c1b.Close()
	defer c2a.Close()
	defer c2b.Close()

	go Pipe(c1b, c2a)

	done := make(chan bool)
	go func() {
		buf := make([]byte, 4)
		_, _ = io.ReadFull(c2b, buf)
		if string(buf) != "PING" {
			t.Errorf("Expected PING, got %s", string(buf))
		}
		_, _ = c2b.Write([]byte("PONG"))
	}()

	go func() {
		_, _ = c1a.Write([]byte("PING"))
		buf := make([]byte, 4)
		_, _ = io.ReadFull(c1a, buf)
		if string(buf) != "PONG" {
			t.Errorf("Expected PONG, got %s", string(buf))
		}
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Pipe timed out")
	}
}
