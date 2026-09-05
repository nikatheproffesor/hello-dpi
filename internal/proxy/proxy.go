package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/speedtest"
)

// Server is the core Hello DPI proxy server
type Server struct {
	Addr         string
	Engine       *dpi.FragmentEngine
	Resolver     *doh.Resolver
	speedtestMux *http.ServeMux
	listener     net.Listener
	bufferPool   sync.Pool
	mu           sync.Mutex
	closed       bool
}

// Config holds configuration options for Server
type Config struct {
	Addr        string
	SplitMode   dpi.SplitMode
	DelayMs     int
	DoHEndpoint string
	EnableDoH   bool
}

// NewServer initializes a new Server
func NewServer(cfg Config) *Server {
	mux := http.NewServeMux()
	speedtest.RegisterHandlers(mux)

	return &Server{
		Addr:         cfg.Addr,
		Engine:       dpi.NewFragmentEngine(cfg.SplitMode, cfg.DelayMs),
		Resolver:     doh.NewResolver(cfg.DoHEndpoint, cfg.EnableDoH),
		speedtestMux: mux,
		bufferPool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, 32*1024) // 32KB buffer for high throughput
				return &b
			},
		},
	}
}

// Start listens for incoming connections and serves them
func (s *Server) Start() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.Addr, err)
	}
	s.mu.Lock()
	s.listener = l
	s.mu.Unlock()

	log.Printf("[Hello DPI] Proxy listening on %s (Dual HTTP & SOCKS5)", s.Addr)

	for {
		clientConn, err := l.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			log.Printf("[Hello DPI] Accept error: %v", err)
			continue
		}

		go s.handleConnection(clientConn)
	}
}

// Close gracefully stops the proxy listener
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConnection auto-detects between HTTP CONNECT / plain HTTP and SOCKS5
func (s *Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	reader := bufio.NewReader(clientConn)
	firstByte, err := reader.Peek(1)
	if err != nil {
		return
	}

	// SOCKS5 starts with byte 0x05
	if firstByte[0] == 0x05 {
		s.handleSOCKS5(clientConn, reader)
		return
	}

	// Otherwise handle as HTTP / HTTPS CONNECT
	s.handleHTTP(clientConn, reader)
}

// handleHTTP handles HTTP CONNECT (HTTPS), regular HTTP proxy requests, and internal speedtest
func (s *Server) handleHTTP(clientConn net.Conn, reader *bufio.Reader) {
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	// Intercept local Speedtest endpoints
	if req.Method != http.MethodConnect && (req.URL.Path == "/speedtest" || strings.HasPrefix(req.URL.Path, "/api/speedtest")) {
		w := newConnResponseWriter(clientConn)
		s.speedtestMux.ServeHTTP(w, req)
		w.flushHeaders()
		return
	}

	targetHost := req.Host
	if targetHost == "" {
		targetHost = req.URL.Host
	}

	// Ensure port is present
	host, port, err := net.SplitHostPort(targetHost)
	if err != nil {
		host = targetHost
		if req.Method == http.MethodConnect {
			port = "443"
		} else {
			port = "80"
		}
	}

	// Resolve target using DoH or system DNS
	resolvedIP, err := s.Resolver.Resolve(context.Background(), host)
	if err != nil {
		resolvedIP = host // fallback to unresolved
	}
	destAddr := net.JoinHostPort(resolvedIP, port)

	// Dial target server
	targetConn, err := net.DialTimeout("tcp", destAddr, 10*time.Second)
	if err != nil {
		if req.Method == http.MethodConnect {
			_, _ = clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		}
		return
	}
	defer targetConn.Close()

	if req.Method == http.MethodConnect {
		// Respond 200 OK to the client
		_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if err != nil {
			return
		}

		// Read complete initial packet (e.g. complete TLS ClientHello via io.ReadFull)
		initialPayload, err := readInitialPayload(reader)
		if err == nil && len(initialPayload) > 0 {
			// Fragment and transmit the initial TLS handshake
			if err := s.Engine.SendFragmented(targetConn, initialPayload); err != nil {
				return
			}
		}

		// Stream bidirectional data at full line rate
		s.pipe(clientConn, targetConn)
	} else {
		// Plain HTTP: re-serialize request and fragment it
		var reqBuf []byte
		var b strings.Builder
		_ = req.Write(&b)
		reqBuf = []byte(b.String())

		if err := s.Engine.SendFragmented(targetConn, reqBuf); err != nil {
			return
		}

		s.pipe(clientConn, targetConn)
	}
}

// handleSOCKS5 implements RFC 1928 SOCKS5 protocol with DPI fragmentation and QUIC block
func (s *Server) handleSOCKS5(clientConn net.Conn, reader *bufio.Reader) {
	// 1. Negotiation
	ver, err := reader.ReadByte()
	if err != nil || ver != 0x05 {
		return
	}
	nmethods, err := reader.ReadByte()
	if err != nil {
		return
	}
	methods := make([]byte, nmethods)
	if _, err := io.ReadFull(reader, methods); err != nil {
		return
	}

	// Respond with NO AUTHENTICATION REQUIRED (0x00)
	if _, err := clientConn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	// 2. Request details
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}

	cmd := header[1]
	// If client asks for UDP ASSOCIATE (0x03) e.g. for QUIC / HTTP-3:
	// Reject with 0x07 (Command not supported) so client automatically falls back to TCP + TLS!
	if cmd != 0x01 { // 0x01 = CONNECT
		_, _ = clientConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Command not supported
		return
	}

	atyp := header[3]
	var targetHost string
	switch atyp {
	case 0x01: // IPv4
		ip := make([]byte, 4)
		if _, err := io.ReadFull(reader, ip); err != nil {
			return
		}
		targetHost = net.IP(ip).String()
	case 0x03: // Domain name
		dLen, err := reader.ReadByte()
		if err != nil {
			return
		}
		domain := make([]byte, dLen)
		if _, err := io.ReadFull(reader, domain); err != nil {
			return
		}
		targetHost = string(domain)
	case 0x04: // IPv6
		ip := make([]byte, 16)
		if _, err := io.ReadFull(reader, ip); err != nil {
			return
		}
		targetHost = net.IP(ip).String()
	default:
		_, _ = clientConn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, portBytes); err != nil {
		return
	}
	port := strconv.Itoa(int(portBytes[0])<<8 | int(portBytes[1]))

	// Resolve target
	resolvedIP, err := s.Resolver.Resolve(context.Background(), targetHost)
	if err != nil {
		resolvedIP = targetHost
	}
	destAddr := net.JoinHostPort(resolvedIP, port)

	targetConn, err := net.DialTimeout("tcp", destAddr, 10*time.Second)
	if err != nil {
		_, _ = clientConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Host unreachable
		return
	}
	defer targetConn.Close()

	// SOCKS5 success reply
	_, _ = clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Read complete initial payload from client (e.g. TLS ClientHello)
	initialPayload, err := readInitialPayload(reader)
	if err == nil && len(initialPayload) > 0 {
		_ = s.Engine.SendFragmented(targetConn, initialPayload)
	}

	s.pipe(clientConn, targetConn)
}

// readInitialPayload guarantees reading the entire TLS ClientHello packet even if chunked by OS
func readInitialPayload(reader *bufio.Reader) ([]byte, error) {
	hdr, err := reader.Peek(5)
	if err != nil {
		buf := make([]byte, 2048)
		n, err := reader.Read(buf)
		return buf[:n], err
	}

	// Check if this is a TLS Record (0x16 Handshake)
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

	buf := make([]byte, 8192)
	n, err := reader.Read(buf)
	return buf[:n], err
}

// pipe streams traffic bidirectionally with zero-copy buffer pooling
func (s *Server) pipe(src, dst net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	cp := func(to, from net.Conn) {
		defer wg.Done()
		bufPtr := s.bufferPool.Get().(*[]byte)
		defer s.bufferPool.Put(bufPtr)

		_, _ = io.CopyBuffer(to, from, *bufPtr)
		if tc, ok := to.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}

	go cp(dst, src)
	go cp(src, dst)

	wg.Wait()
}

// connResponseWriter implements http.ResponseWriter and http.Flusher directly over net.Conn
type connResponseWriter struct {
	conn        net.Conn
	headers     http.Header
	wroteHeader bool
	status      int
}

func newConnResponseWriter(c net.Conn) *connResponseWriter {
	return &connResponseWriter{
		conn:    c,
		headers: make(http.Header),
		status:  http.StatusOK,
	}
}

func (w *connResponseWriter) Header() http.Header {
	return w.headers
}

func (w *connResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.status = statusCode
		w.wroteHeader = true
		w.flushHeaders()
	}
}

func (w *connResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(data)
}

func (w *connResponseWriter) Flush() {
	// TCP socket flushes automatically when TCP_NODELAY is enabled
}

func (w *connResponseWriter) flushHeaders() {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", w.status, http.StatusText(w.status)))
	for k, vv := range w.headers {
		for _, v := range vv {
			sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	sb.WriteString("\r\n")
	_, _ = w.conn.Write([]byte(sb.String()))
}
