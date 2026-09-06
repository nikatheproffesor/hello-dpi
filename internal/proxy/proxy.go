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

// Start begins accepting connections on the configured address
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

// handleConnection discriminates between SOCKS5 and HTTP protocols
func (s *Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Initial handshake deadline to avoid socket slowloris starvation
	_ = clientConn.SetDeadline(time.Now().Add(15 * time.Second))

	reader := bufio.NewReader(clientConn)
	firstByte, err := reader.Peek(1)
	if err != nil {
		return
	}

	// Reset deadline for streaming phase
	_ = clientConn.SetDeadline(time.Time{})

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
	if control.IsControlPath(req.Method, req.URL.Path) {
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
		// Respond 200 Connection Established to the client
		if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
		_ = s.Orchestrator.HandleTunnel(clientConn, reader, host, port)
	} else {
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

	// 0x00 = NO AUTHENTICATION REQUIRED
	if _, err := clientConn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}

	// 2. Request details
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return
	}

	cmd := header[1]
	// If client requests UDP ASSOCIATE (0x03) for QUIC: reject with 0x07 (Command not supported)
	// forcing client to fallback cleanly to TCP + TLS!
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

	// SOCKS5 success reply
	if _, err := clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}

	_ = s.Orchestrator.HandleTunnel(clientConn, reader, targetHost, port)
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
