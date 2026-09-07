package tun

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

// buildSyntheticIPv4TCP constructs an IPv4 TCP packet
func buildSyntheticIPv4TCP(srcIP, dstIP net.IP, srcPort, dstPort uint16, payload []byte) []byte {
	ipHeaderLen := 20
	tcpHeaderLen := 20
	totalLen := ipHeaderLen + tcpHeaderLen + len(payload)

	pkt := make([]byte, totalLen)
	// IPv4 Header
	pkt[0] = 0x45 // Version 4, IHL 5 (20 bytes)
	binary.BigEndian.PutUint16(pkt[2:4], uint16(totalLen))
	pkt[8] = 64 // TTL
	pkt[9] = 6  // Protocol TCP
	copy(pkt[12:16], srcIP.To4())
	copy(pkt[16:20], dstIP.To4())

	// TCP Header
	offset := ipHeaderLen
	binary.BigEndian.PutUint16(pkt[offset:offset+2], srcPort)
	binary.BigEndian.PutUint16(pkt[offset+2:offset+4], dstPort)
	pkt[offset+12] = 0x50 // Data offset: 5 (20 bytes)
	pkt[offset+13] = 0x18 // Flags: PSH | ACK

	// Payload
	copy(pkt[offset+tcpHeaderLen:], payload)
	return pkt
}

func TestTransparentTUNEngine(t *testing.T) {
	mockDev := NewMockDevice("hellotun0", 1500)
	engine := NewEngine(mockDev, nil)

	if err := engine.Start(); err != nil {
		t.Fatalf("engine.Start failed: %v", err)
	}
	defer engine.Stop()

	srcIP := net.ParseIP("192.168.1.100")
	dstIP := net.ParseIP("162.159.138.232") // Discord IP
	payload := []byte{0x16, 0x03, 0x01, 0x00, 0x10, 0x01}

	pkt := buildSyntheticIPv4TCP(srcIP, dstIP, 54321, 443, payload)
	mockDev.InjectPacket(pkt)

	// Wait briefly for packet pump
	time.Sleep(50 * time.Millisecond)

	stats := engine.GetStats()
	if stats.PacketsIn != 1 {
		t.Errorf("expected 1 packet in, got %d", stats.PacketsIn)
	}
	if stats.TCPIntercept != 1 {
		t.Errorf("expected 1 TCP intercepted, got %d", stats.TCPIntercept)
	}
}
