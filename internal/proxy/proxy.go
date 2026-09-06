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

	"github.com/hellodpi/hellodpi/internal/doctor"
	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/rules"
	"github.com/hellodpi/hellodpi/internal/speedtest"
)

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"::1/128",        // IPv6 loopback
		"10.0.0.0/8",     // RFC1918 (GSB WiFi / KYK dorm networks)
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateOrLocalIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// isDirectPassThrough checks if a host should bypass DPI fragmentation and DoH (GSB WiFi / Captive Portals)
func isDirectPassThrough(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	if h == "localhost" || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".lan") || strings.HasSuffix(h, ".home") {
		return true
	}
	if strings.Contains(h, "gsb.gov.tr") ||
		strings.Contains(h, "kyk.gov.tr") ||
		h == "captive.apple.com" ||
		h == "connectivitycheck.gstatic.com" ||
		h == "connectivitycheck.android.com" ||
		h == "msftconnecttest.com" ||
		h == "ipv6.msftconnecttest.com" {
		return true
	}
	return isPrivateOrLocalIP(h)
}

// Server is the core Hello DPI proxy server
type Server struct {
	Addr         string
	Engine       *dpi.FragmentEngine
	Resolver     *doh.Resolver
	Rules        *rules.Engine
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
	doctor.RegisterHandlers(mux)

	return &Server{
		Addr:         cfg.Addr,
		Engine:       dpi.NewFragmentEngine(cfg.SplitMode, cfg.DelayMs),
		Resolver:     doh.NewResolver(cfg.DoHEndpoint, cfg.EnableDoH),
		Rules:        rules.NewEngine(),
		speedtestMux: mux,
		bufferPool: sync.Pool{
			New: func() interface{} {
				b := make([]byte, 32*1024) // 32KB buffer for high throughput
				return &b
			},
		},
	}
}

// UpdateEngineConfig dynamically reconfigures the fragment engine on the fly
func (s *Server) UpdateEngineConfig(mode dpi.SplitMode, splitOffset int, delayMs int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Engine = &dpi.FragmentEngine{
		Mode:         mode,
		ChunkDelay:   time.Duration(delayMs) * time.Millisecond,
		CustomOffset: splitOffset,
	}
	log.Printf("[Hello DPI] Fragment engine auto-tuned: mode=%s, splitPos=%d, delay=%dms", mode, splitOffset, delayMs)
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

// bufferedConn guarantees that unread bytes buffered in bufio.Reader are never lost
type bufferedConn struct {
	r io.Reader
	net.Conn
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

// handleConnection auto-detects between HTTP CONNECT / plain HTTP and SOCKS5
func (s *Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// 15-second handshake deadline to prevent slowloris socket exhaustion
	_ = clientConn.SetDeadline(time.Now().Add(15 * time.Second))

	reader := bufio.NewReader(clientConn)
	firstByte, err := reader.Peek(1)
	if err != nil {
		return
	}

	// Clear deadline for streaming phase
	_ = clientConn.SetDeadline(time.Time{})

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

	// Intercept local Speedtest and Doctor endpoints
	pathLower := strings.ToLower(req.URL.Path)
	if req.Method != http.MethodConnect && (pathLower == "/speedtest" || strings.HasPrefix(pathLower, "/speedtest/") ||
		strings.HasPrefix(pathLower, "/api/speedtest") || pathLower == "/doctor" ||
		strings.HasPrefix(pathLower, "/doctor/") || strings.HasPrefix(pathLower, "/api/doctor")) {
		w := newConnResponseWriter(clientConn)
		s.speedtestMux.ServeHTTP(w, req)
		return
	}

	if req.URL.Path == "/favicon.ico" {
		w := newConnResponseWriter(clientConn)
		w.WriteHeader(http.StatusNoContent)
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

	// Anti-Poison Protection: if the client was tricked by poisoned ISP DNS into connecting
	// to 195.175.254.x (BTK court order block page), intercept it, read the real SNI,
	// and connect to the real server via DoH!
	if req.Method == http.MethodConnect && strings.HasPrefix(host, "195.175.254.") {
		s.handlePoisonedConnect(clientConn, reader, port)
		return
	}

	directPass := isDirectPassThrough(host) || (s.Rules != nil && s.Rules.Evaluate(host) == rules.ActionDirect)

	// Resolve target using DoH or system DNS
	var resolvedIP string
	if directPass {
		// Captive portal / local network: use OS DNS directly
		ips, err := net.DefaultResolver.LookupHost(context.Background(), host)
		if err == nil && len(ips) > 0 {
			resolvedIP = ips[0]
		} else {
			resolvedIP = host
		}
	} else {
		resolvedIP, err = s.Resolver.Resolve(context.Background(), host)
		if err != nil {
			resolvedIP = host
		}
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

	// Wrap clientConn with bufferedConn so any bytes in reader are drained first!
	clientBuffered := &bufferedConn{r: reader, Conn: clientConn}

	if req.Method == http.MethodConnect {
		// Respond 200 OK to the client
		_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		if err != nil {
			return
		}

		if directPass {
			// Direct pass-through for GSB WiFi / local networks: zero fragmentation
			s.pipe(clientBuffered, targetConn)
			return
		}

		// Read complete initial packet (e.g. complete TLS ClientHello via io.ReadFull)
		initialPayload, err := readInitialPayload(reader)
		if err == nil && len(initialPayload) > 0 {
			// Fragment and transmit the initial TLS handshake with advanced evasion
			if err := s.Engine.SendAdvancedEvasion(targetConn, initialPayload); err != nil {
				return
			}
		}

		// Stream bidirectional data at full line rate with buffer preservation for WebSockets (w2g.tv)
		s.pipe(clientBuffered, targetConn)
	} else {
		// Plain HTTP or WebSocket Upgrade
		isWS := strings.EqualFold(req.Header.Get("Upgrade"), "websocket")

		if isWS || directPass {
			_ = req.Write(targetConn)
			s.pipe(clientBuffered, targetConn)
			return
		}

		// Plain HTTP: serialize and fragment request
		var b strings.Builder
		_ = req.Write(&b)
		reqBuf := []byte(b.String())

		if err := s.Engine.SendAdvancedEvasion(targetConn, reqBuf); err != nil {
			return
		}

		s.pipe(clientBuffered, targetConn)
	}
}

// handlePoisonedConnect recovers connections hijacked by Turkish ISP DNS to 195.175.254.x
func (s *Server) handlePoisonedConnect(clientConn net.Conn, reader *bufio.Reader, port string) {
	_, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		return
	}

	initialPayload, err := readInitialPayload(reader)
	if err != nil || len(initialPayload) == 0 {
		return
	}

	info := dpi.ParsePacket(initialPayload)
	realHost := info.Host
	if realHost == "" {
		return
	}

	realIP, err := s.Resolver.Resolve(context.Background(), realHost)
	if err != nil || realIP == "" || strings.HasPrefix(realIP, "195.175.254.") {
		return
	}

	destAddr := net.JoinHostPort(realIP, port)
	targetConn, err := net.DialTimeout("tcp", destAddr, 10*time.Second)
	if err != nil {
		return
	}
	defer targetConn.Close()

	if err := s.Engine.SendAdvancedEvasion(targetConn, initialPayload); err != nil {
		return
	}

	clientBuffered := &bufferedConn{r: reader, Conn: clientConn}
	s.pipe(clientBuffered, targetConn)
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

	directPass := isDirectPassThrough(targetHost) || (s.Rules != nil && s.Rules.Evaluate(targetHost) == rules.ActionDirect)

	// Resolve target
	var resolvedIP string
	if directPass {
		ips, err := net.DefaultResolver.LookupHost(context.Background(), targetHost)
		if err == nil && len(ips) > 0 {
			resolvedIP = ips[0]
		} else {
			resolvedIP = targetHost
		}
	} else {
		resolvedIP, err = s.Resolver.Resolve(context.Background(), targetHost)
		if err != nil {
			resolvedIP = targetHost
		}
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

	clientBuffered := &bufferedConn{r: reader, Conn: clientConn}

	if directPass {
		s.pipe(clientBuffered, targetConn)
		return
	}

	// Read complete initial payload from client (e.g. TLS ClientHello)
	initialPayload, err := readInitialPayload(reader)
	if err == nil && len(initialPayload) > 0 {
		_ = s.Engine.SendAdvancedEvasion(targetConn, initialPayload)
	}

	s.pipe(clientBuffered, targetConn)
}

// isProxyAddr checks if the target matches the proxy's own address
func (s *Server) isProxyAddr(host, port string) bool {
	sHost, sPort, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return false
	}
	if port == sPort && (host == sHost || host == "127.0.0.1" || host == "localhost") {
		return true
	}
	return false
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

// pipe streams traffic bidirectionally with zero-copy buffer pooling and clean half-close
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

func (w *connResponseWriter) Flush() {
	// TCP socket flushes automatically when TCP_NODELAY is enabled
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
