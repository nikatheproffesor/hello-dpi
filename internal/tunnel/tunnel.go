package tunnel

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"sync"
)

// DefaultBufferPool provides reusable 32KB buffers for high-throughput zero-copy streaming
var DefaultBufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

// BufferedConn wraps a net.Conn with a buffered io.Reader so that any pre-read
// bytes (e.g. from protocol sniffing or peek) are drained completely before reading
// from the raw underlying socket.
type BufferedConn struct {
	r io.Reader
	net.Conn
}

// NewBufferedConn creates a BufferedConn wrapping the reader and raw socket.
func NewBufferedConn(r io.Reader, conn net.Conn) *BufferedConn {
	return &BufferedConn{
		r:    r,
		Conn: conn,
	}
}

func (b *BufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

// Pipe streams bidirectional data between two connections with buffer reuse and clean half-close.
func Pipe(src, dst net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	cp := func(to, from net.Conn) {
		defer wg.Done()
		bufPtr := DefaultBufferPool.Get().(*[]byte)
		defer DefaultBufferPool.Put(bufPtr)

		_, _ = io.CopyBuffer(to, from, *bufPtr)
		if tc, ok := to.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}

	go cp(dst, src)
	go cp(src, dst)

	wg.Wait()
}

// MaxInitialPayloadSize is the maximum allowed size for the initial packet (standard TLS record is 16KB + 5 bytes header)
const MaxInitialPayloadSize = 16384 + 5

// ReadInitialPayload safely extracts the complete initial packet (e.g. TLS ClientHello)
// from the buffered reader without memory explosion risks or incomplete chunk issues.
func ReadInitialPayload(reader *bufio.Reader) ([]byte, error) {
	hdr, err := reader.Peek(5)
	if err != nil {
		buf := make([]byte, 2048)
		n, err := reader.Read(buf)
		return buf[:n], err
	}

	// TLS Handshake Record check (0x16)
	if hdr[0] == 0x16 {
		recLen := int(binary.BigEndian.Uint16(hdr[3:5]))
		// Valid TLS record length check (up to 16KB)
		if recLen > 0 && recLen <= 16384 {
			totalLen := 5 + recLen
			buf := make([]byte, totalLen)
			_, err := io.ReadFull(reader, buf)
			return buf, err
		}
	}

	// HTTP or generic protocol initial burst
	buf := make([]byte, 8192)
	n, err := reader.Read(buf)
	return buf[:n], err
}
