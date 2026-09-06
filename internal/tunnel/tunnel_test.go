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
	pipeReader, pipeWriter := net.Pipe()
	defer pipeReader.Close()
	defer pipeWriter.Close()

	go func() {
		_, _ = pipeWriter.Write([]byte("WORLD"))
	}()

	// Put "HELLO " in a buffer, and socket has "WORLD"
	preBuf := bytes.NewReader([]byte("HELLO "))
	reader := bufio.NewReader(io.MultiReader(preBuf, pipeReader))

	bc := NewBufferedConn(reader, pipeReader)

	data := make([]byte, 11)
	n, err := io.ReadFull(bc, data)
	if err != nil {
		t.Fatalf("ReadFull failed: %v", err)
	}
	if n != 11 || string(data) != "HELLO WORLD" {
		t.Fatalf("Expected 'HELLO WORLD', got '%s'", string(data))
	}
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
