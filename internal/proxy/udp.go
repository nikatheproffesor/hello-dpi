package proxy

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/quic"
)

// UDPRelay manages SOCKS5 UDP ASSOCIATE sessions, inspecting QUIC handshakes
// and providing zero-overhead routing for Discord WebRTC / game traffic.
type UDPRelay struct {
	mu         sync.RWMutex
	udpConn    *net.UDPConn
	mangler    *quic.Mangler
	closed     bool
	clientAddr *net.UDPAddr
}

// NewUDPRelay binds a local UDP relay socket for SOCKS5 UDP ASSOCIATE
func NewUDPRelay() (*UDPRelay, error) {
	lAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		return nil, err
	}

	relay := &UDPRelay{
		udpConn: conn,
		mangler: quic.NewMangler(),
	}

	return relay, nil
}

// LocalAddr returns the local bound address of the UDP relay
func (r *UDPRelay) LocalAddr() *net.UDPAddr {
	return r.udpConn.LocalAddr().(*net.UDPAddr)
}

// Close terminates the UDP relay socket
func (r *UDPRelay) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.udpConn != nil {
		return r.udpConn.Close()
	}
	return nil
}

// Serve runs the UDP relay packet processing loop.
// It stops when stopSignal is closed or an unrecoverable error occurs.
func (r *UDPRelay) Serve(stopSignal <-chan struct{}) {
	buf := make([]byte, 65535)

	// Outbound remote connections map keyed by remote address string
	type activeConn struct {
		conn       *net.UDPConn
		lastActive time.Time
	}
	remoteConns := make(map[string]*activeConn)
	var remoteMu sync.Mutex

	// Background idle timeout sweeper (2 minutes)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stopSignal:
				return
			case <-ticker.C:
				now := time.Now()
				remoteMu.Lock()
				for k, ac := range remoteConns {
					if now.Sub(ac.lastActive) > 2*time.Minute {
						_ = ac.conn.Close()
						delete(remoteConns, k)
					}
				}
				remoteMu.Unlock()
			}
		}
	}()

	defer func() {
		remoteMu.Lock()
		for _, ac := range remoteConns {
			_ = ac.conn.Close()
		}
		remoteMu.Unlock()
		_ = r.Close()
	}()

	go func() {
		<-stopSignal
		_ = r.Close()
	}()

	for {
		n, srcAddr, err := r.udpConn.ReadFromUDP(buf)
		if err != nil {
			r.mu.RLock()
			closed := r.closed
			r.mu.RUnlock()
			if closed {
				return
			}
			continue
		}

		r.mu.Lock()
		r.clientAddr = srcAddr
		r.mu.Unlock()

		if n < 10 {
			continue
		}

		// SOCKS5 UDP Header:
		// [RSV 2B][FRAG 1B][ATYP 1B][DST.ADDR][DST.PORT][DATA]
		atyp := buf[3]
		var destHost string
		var offset int

		switch atyp {
		case 0x01: // IPv4
			if n < 10 {
				continue
			}
			destHost = net.IP(buf[4:8]).String()
			offset = 8
		case 0x03: // Domain name
			dLen := int(buf[4])
			if n < 5+dLen+2 {
				continue
			}
			destHost = string(buf[5 : 5+dLen])
			offset = 5 + dLen
		case 0x04: // IPv6
			if n < 22 {
				continue
			}
			destHost = net.IP(buf[4:20]).String()
			offset = 20
		default:
			continue
		}

		destPort := int(binary.BigEndian.Uint16(buf[offset : offset+2]))
		offset += 2
		payload := buf[offset:n]

		destStr := net.JoinHostPort(destHost, strconv.Itoa(destPort))
		destUDPAddr, err := net.ResolveUDPAddr("udp", destStr)
		if err != nil {
			continue
		}

		// Look up or establish outbound UDP socket to the destination
		remoteMu.Lock()
		ac, exists := remoteConns[destStr]
		if !exists {
			outConn, err := net.DialUDP("udp", nil, destUDPAddr)
			if err != nil {
				remoteMu.Unlock()
				continue
			}
			ac = &activeConn{
				conn:       outConn,
				lastActive: time.Now(),
			}
			remoteConns[destStr] = ac

			// Spawn response reader for this remote endpoint
			go r.pipeRemoteToClient(outConn, destHost, destPort, atyp, func() {
				// Update last active time on receive
				remoteMu.Lock()
				if c, ok := remoteConns[destStr]; ok {
					c.lastActive = time.Now()
				}
				remoteMu.Unlock()
			})
		} else {
			ac.lastActive = time.Now()
		}
		outConn := ac.conn
		remoteMu.Unlock()

		// Apply QUIC packet mangling or zero-latency voice forwarding
		packets := r.mangler.ProcessOutbound(destUDPAddr, payload)
		for _, pkt := range packets {
			_, _ = outConn.Write(pkt)
		}
	}
}

// pipeRemoteToClient receives server responses, adds SOCKS5 UDP header, and sends back to client
func (r *UDPRelay) pipeRemoteToClient(remoteConn *net.UDPConn, host string, port int, atyp byte, onActive func()) {
	respBuf := make([]byte, 65535)
	for {
		n, err := remoteConn.Read(respBuf)
		if err != nil {
			return
		}
		if onActive != nil {
			onActive()
		}

		r.mu.RLock()
		client := r.clientAddr
		r.mu.RUnlock()

		if client == nil {
			continue
		}

		// Encapsulate response: [RSV 2B][FRAG 1B][ATYP 1B][ADDR][PORT][DATA]
		var hdr []byte
		if atyp == 0x01 {
			ip := net.ParseIP(host).To4()
			if ip == nil {
				ip = net.IPv4zero
			}
			hdr = make([]byte, 10)
			hdr[3] = 0x01
			copy(hdr[4:8], ip)
			binary.BigEndian.PutUint16(hdr[8:10], uint16(port))
		} else {
			dBytes := []byte(host)
			hdr = make([]byte, 7+len(dBytes))
			hdr[3] = 0x03
			hdr[4] = byte(len(dBytes))
			copy(hdr[5:5+len(dBytes)], dBytes)
			binary.BigEndian.PutUint16(hdr[5+len(dBytes):], uint16(port))
		}

		// Safe concatenation: allocate a new buffer to avoid hdr backing array corruption
		fullResp := make([]byte, len(hdr)+n)
		copy(fullResp, hdr)
		copy(fullResp[len(hdr):], respBuf[:n])
		_, _ = r.udpConn.WriteToUDP(fullResp, client)
	}
}

// HandleSOCKS5UDPAssociate responds to SOCKS5 UDP ASSOCIATE and launches relay loop
func HandleSOCKS5UDPAssociate(clientConn net.Conn, tcpReader io.Reader) error {
	relay, err := NewUDPRelay()
	if err != nil {
		// General SOCKS server failure
		_, _ = clientConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return err
	}

	boundAddr := relay.LocalAddr()
	ip4 := boundAddr.IP.To4()
	if ip4 == nil {
		ip4 = net.IPv4(127, 0, 0, 1)
	}

	// SOCKS5 reply: 0x05 (ver), 0x00 (success), 0x00 (rsv), 0x01 (ipv4), [4B IP], [2B PORT]
	reply := make([]byte, 10)
	reply[0] = 0x05
	reply[1] = 0x00 // Success
	reply[2] = 0x00
	reply[3] = 0x01 // IPv4
	copy(reply[4:8], ip4)
	binary.BigEndian.PutUint16(reply[8:10], uint16(boundAddr.Port))

	if _, err := clientConn.Write(reply); err != nil {
		_ = relay.Close()
		return err
	}

	stopCh := make(chan struct{})
	go relay.Serve(stopCh)

	// Keep TCP connection alive until client disconnects
	go func() {
		buf := make([]byte, 128)
		for {
			_ = clientConn.SetDeadline(time.Now().Add(60 * time.Second))
			_, err := tcpReader.Read(buf)
			if err != nil {
				close(stopCh)
				_ = relay.Close()
				return
			}
		}
	}()

	log.Printf("[Hello DPI] SOCKS5 UDP Associate established on :%d (QUIC & Voice Active)", boundAddr.Port)
	return nil
}
