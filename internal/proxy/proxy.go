package proxy

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/control"
	"github.com/hellodpi/hellodpi/internal/dns"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/engine"
	"github.com/hellodpi/hellodpi/internal/rules"
)

// Server is the clean L7 proxy server focusing strictly on accepting connections,
// parsing SOCKS5 / HTTP protocols, and delegating to the orchestration engine.
type Server struct {
	Addr         string
	Orchestrator *engine.Orchestrator
	Rules        *rules.Engine
	Resolver     *dns.Resolver
	Control      *control.Server
	listener     net.Listener
	mu           sync.Mutex
	closed       bool
	activeWg     sync.WaitGroup
}

// Config holds initialization parameters for the Server
type Config struct {
	Addr        string
	SplitMode   dpi.SplitMode
	DelayMs     int
	DoHEndpoint string
	EnableDoH   bool
}

// NewServer initializes a new clean proxy Server
func NewServer(cfg Config) *Server {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8080"
	}

	engCfg := engine.Config{
		StrategyName: string(cfg.SplitMode),
		SplitOffset:  5,
		DelayMs:      cfg.DelayMs,
		DoHEndpoint:  cfg.DoHEndpoint,
		EnableDoH:    cfg.EnableDoH,
	}

	orch := engine.NewOrchestrator(engCfg)
	ctrl := control.NewServer()

	return &Server{
		Addr:         cfg.Addr,
		Orchestrator: orch,
		Rules:        orch.Rules,
		Resolver:     orch.Resolver,
		Control:      ctrl,
	}
}

// UpdateEngineConfig dynamically reconfigures the bypass strategy
func (s *Server) UpdateEngineConfig(mode dpi.SplitMode, splitOffset int, delayMs int) {
	s.Orchestrator.UpdateStrategy(string(mode), splitOffset, delayMs)
}

// Listen creates the TCP listener on s.Addr synchronously so the caller can guarantee port availability
func (s *Server) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = false
	if s.listener != nil {
		return nil
	}

	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.Addr, err)
	}
	s.listener = l
	return nil
}

// Serve accepts incoming connections on the pre-bound listener
func (s *Server) Serve() error {
	s.mu.Lock()
	l := s.listener
	s.mu.Unlock()

	if l == nil {
		if err := s.Listen(); err != nil {
			return err
		}
		s.mu.Lock()
		l = s.listener
		s.mu.Unlock()
	}

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

		s.activeWg.Add(1)
		go func(c net.Conn) {
			defer s.activeWg.Done()
			s.handleConnection(c)
		}(clientConn)
	}
}

// Start begins accepting connections (Listen + Serve)
func (s *Server) Start() error {
	if err := s.Listen(); err != nil {
		return err
	}
	return s.Serve()
}

// Close gracefully stops the proxy listener and marks closed
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	l := s.listener
	s.listener = nil
	s.mu.Unlock()

	if l != nil {
		return l.Close()
	}
	return nil
}

// handleConnection discriminates between SOCKS5 and HTTP protocols
func (s *Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Handshake deadline protects against socket slowloris starvation
	_ = clientConn.SetDeadline(time.Now().Add(15 * time.Second))

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

	// Otherwise process as HTTP / HTTPS CONNECT
	s.handleHTTP(clientConn, reader)
}

// handleHTTP parses HTTP requests, routes local management endpoints to control,
// and delegates HTTPS CONNECT / plain HTTP to the orchestrator.
func (s *Server) handleHTTP(clientConn net.Conn, reader *bufio.Reader) {
	req, err := http.ReadRequest(reader)
	if err != nil {
		return
	}

	// Route local management endpoints (/doctor, /speedtest, /api/status) to control server
	// only if the request genuinely targets loopback, avoiding hijacking external sites.
	if control.IsControlRequest(req) {
		w := newConnResponseWriter(clientConn)
		s.Control.ServeHTTP(w, req)
		return
	}

	targetHost := req.Host
	if targetHost == "" {
		targetHost = req.URL.Host
	}

	host, port, err := net.SplitHostPort(targetHost)
	if err != nil {
		host = targetHost
		if req.Method == http.MethodConnect {
			port = "443"
		} else {
			port = "80"
		}
	}

	// Self-loop prevention: abort if target is proxy's own address
	if s.isProxyLoop(host, port) {
		return
	}

	if req.Method == http.MethodConnect {
		err := s.Orchestrator.HandleTunnel(clientConn, reader, host, port, func() error {
			// Clear deadline for streaming phase
			_ = clientConn.SetDeadline(time.Time{})
			_, wErr := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			return wErr
		})
		if err != nil {
			_, _ = clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		}
	} else {
		_ = clientConn.SetDeadline(time.Time{})
		_ = s.Orchestrator.HandleHTTP(clientConn, reader, req, host, port)
	}
}

// handleSOCKS5 implements RFC 1928 SOCKS5 protocol handshake
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

	// Verify client offers NO AUTHENTICATION REQUIRED (0x00)
	hasNoAuth := false
	for _, m := range methods {
		if m == 0x00 {
			hasNoAuth = true
			break
		}
	}
	if !hasNoAuth {
		_, _ = clientConn.Write([]byte{0x05, 0xFF}) // No acceptable methods
		return
	}

	if _, err := clientConn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	// 2. Request details
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}

	// RFC 1928: header[0] must be VER=0x05, header[2] must be RSV=0x00
	if header[0] != 0x05 || header[2] != 0x00 {
		_, _ = clientConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // General SOCKS failure
		return
	}

	cmd := header[1]
	if cmd == 0x03 { // 0x03 = UDP ASSOCIATE (QUIC / Discord Voice)
		_ = clientConn.SetDeadline(time.Time{})
		if err := HandleSOCKS5UDPAssociate(clientConn, reader); err != nil {
			log.Printf("[Hello DPI] SOCKS5 UDP Associate error: %v", err)
		}
		return
	}
	if cmd != 0x01 { // 0x01 = CONNECT
		_, _ = clientConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
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

	// Self-loop prevention
	if s.isProxyLoop(targetHost, port) {
		_, _ = clientConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	err = s.Orchestrator.HandleTunnel(clientConn, reader, targetHost, port, func() error {
		_ = clientConn.SetDeadline(time.Time{})
		_, wErr := clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return wErr
	})
	if err != nil {
		_, _ = clientConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Host unreachable
	}
}

func (s *Server) isProxyLoop(host, port string) bool {
	sHost, sPort, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return false
	}
	if port == sPort && (host == sHost || host == "127.0.0.1" || host == "localhost") {
		return true
	}
	return false
}

// connResponseWriter adapts net.Conn to http.ResponseWriter for local control endpoints
type connResponseWriter struct {
	conn          net.Conn
	headers       http.Header
	headerFlushed bool
	status        int
}

func newConnResponseWriter(c net.Conn) *connResponseWriter {
	h := make(http.Header)
	h.Set("Connection", "close")
	return &connResponseWriter{
		conn:    c,
		headers: h,
		status:  http.StatusOK,
	}
}

func (w *connResponseWriter) Header() http.Header {
	return w.headers
}

func (w *connResponseWriter) WriteHeader(statusCode int) {
	if !w.headerFlushed {
		w.status = statusCode
		w.flushHeaders()
	}
}

func (w *connResponseWriter) Write(data []byte) (int, error) {
	if !w.headerFlushed {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(data)
}

func (w *connResponseWriter) flushHeaders() {
	if w.headerFlushed {
		return
	}
	w.headerFlushed = true
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
