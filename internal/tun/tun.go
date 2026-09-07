package tun

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/quic"
)

var (
	ErrDeviceClosed   = errors.New("tun device is closed")
	ErrInvalidIPPacket = errors.New("invalid IP packet")
)

// Device represents a cross-platform L3 virtual network interface (Wintun / utun / tun)
type Device interface {
	io.ReadWriteCloser
	Name() string
	MTU() int
}

// Stats tracks packet volume in transparent TUN mode
type Stats struct {
	PacketsIn    uint64 `json:"packets_in"`
	PacketsOut   uint64 `json:"packets_out"`
	BytesIn      uint64 `json:"bytes_in"`
	BytesOut     uint64 `json:"bytes_out"`
	TCPIntercept uint64 `json:"tcp_intercept"`
	UDPIntercept uint64 `json:"udp_intercept"`
}

// Engine operates the transparent network layer packet pump
type Engine struct {
	dev        Device
	mangler    *quic.Mangler
	strategy   dpi.BypassStrategy
	stats      Stats
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

// NewEngine creates a new transparent TUN packet engine
func NewEngine(dev Device, strat dpi.BypassStrategy) *Engine {
	if strat == nil {
		strat, _ = dpi.GetStrategy(string(dpi.SplitTLS))
	}
	return &Engine{
		dev:      dev,
		mangler:  quic.NewMangler(),
		strategy: strat,
		stopChan: make(chan struct{}),
	}
}

// GetStats returns current packet counters
func (e *Engine) GetStats() Stats {
	return Stats{
		PacketsIn:    atomic.LoadUint64(&e.stats.PacketsIn),
		PacketsOut:   atomic.LoadUint64(&e.stats.PacketsOut),
		BytesIn:      atomic.LoadUint64(&e.stats.BytesIn),
		BytesOut:     atomic.LoadUint64(&e.stats.BytesOut),
		TCPIntercept: atomic.LoadUint64(&e.stats.TCPIntercept),
		UDPIntercept: atomic.LoadUint64(&e.stats.UDPIntercept),
	}
}

// Start launches the transparent packet processing loop
func (e *Engine) Start() error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return nil
	}
	if e.dev == nil {
		e.mu.Unlock()
		return fmt.Errorf("tun device not initialized")
	}
	e.running = true
	e.mu.Unlock()

	log.Printf("[Hello DPI TUN] Transparent engine active on interface '%s' (MTU=%d)", e.dev.Name(), e.dev.MTU())
	go e.packetLoop()
	return nil
}

// Stop terminates the transparent engine
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return nil
	}
	e.running = false
	close(e.stopChan)
	if e.dev != nil {
		return e.dev.Close()
	}
	return nil
}

// packetLoop pumps packets between the TUN device and the DPI bypass core
func (e *Engine) packetLoop() {
	buf := make([]byte, 65535)

	for {
		select {
		case <-e.stopChan:
			return
		default:
		}

		n, err := e.dev.Read(buf)
		if err != nil {
			e.mu.RLock()
			running := e.running
			e.mu.RUnlock()
			if !running {
				return
			}
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if n < 20 {
			continue
		}

		atomic.AddUint64(&e.stats.PacketsIn, 1)
		atomic.AddUint64(&e.stats.BytesIn, uint64(n))

		// Process packet
		e.handleIPPacket(buf[:n])
	}
}

// handleIPPacket parses IPv4 header and dispatches TCP/UDP payloads
func (e *Engine) handleIPPacket(pkt []byte) {
	version := (pkt[0] >> 4) & 0x0F
	if version != 4 {
		// IPv6 pass-through or future expansion
		return
	}

	ihl := int(pkt[0]&0x0F) * 4
	if len(pkt) < ihl {
		return
	}

	proto := pkt[9]
	srcIP := net.IP(pkt[12:16])
	dstIP := net.IP(pkt[16:20])

	switch proto {
	case 6: // TCP
		atomic.AddUint64(&e.stats.TCPIntercept, 1)
		e.handleTCP(srcIP, dstIP, pkt[ihl:])
	case 17: // UDP
		atomic.AddUint64(&e.stats.UDPIntercept, 1)
		e.handleUDP(srcIP, dstIP, pkt[ihl:])
	}
}

func (e *Engine) handleTCP(srcIP, dstIP net.IP, tcpPkt []byte) {
	if len(tcpPkt) < 20 {
		return
	}
	srcPort := binary.BigEndian.Uint16(tcpPkt[0:2])
	dstPort := binary.BigEndian.Uint16(tcpPkt[2:4])
	dataOffset := int((tcpPkt[12] >> 4) & 0x0F) * 4
	if len(tcpPkt) <= dataOffset {
		return
	}

	payload := tcpPkt[dataOffset:]
	if len(payload) == 0 {
		return
	}

	// If destination port is 443 (HTTPS) and payload is TLS ClientHello (0x16 0x03)
	if (dstPort == 443 || dstPort == 8443) && payload[0] == 0x16 && len(payload) > 5 {
		info := dpi.ParsePacket(payload)
		if info.Type == dpi.TypeTLSClientHello && info.Host != "" {
			log.Printf("[Hello DPI TUN] Transparent TLS intercepted: %s:%d (SNI: %s, %d bytes)",
				dstIP.String(), dstPort, info.Host, len(payload))
		}
	}
	_ = srcPort
}

func (e *Engine) handleUDP(srcIP, dstIP net.IP, udpPkt []byte) {
	if len(udpPkt) < 8 {
		return
	}
	dstPort := int(binary.BigEndian.Uint16(udpPkt[2:4]))
	payload := udpPkt[8:]

	if quic.IsQUICPort(dstPort) && quic.IsQUICInitial(payload) {
		log.Printf("[Hello DPI TUN] Transparent QUIC Initial intercepted to %s:%d (%d bytes)",
			dstIP.String(), dstPort, len(payload))
	}
}
