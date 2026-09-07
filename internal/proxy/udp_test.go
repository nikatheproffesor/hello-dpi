package proxy

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestUDPRelayCreationAndClose(t *testing.T) {
	relay, err := NewUDPRelay()
	if err != nil {
		t.Fatalf("failed to create UDPRelay: %v", err)
	}
	defer relay.Close()

	addr := relay.LocalAddr()
	if addr == nil || addr.Port == 0 {
		t.Errorf("invalid bound local UDP port: %v", addr)
	}
}

func TestSOCKS5UDPAssociateLoop(t *testing.T) {
	// 1. Setup a dummy UDP echo server
	echoAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("resolve UDP addr failed: %v", err)
	}
	echoServer, err := net.ListenUDP("udp", echoAddr)
	if err != nil {
		t.Fatalf("failed to listen UDP echo: %v", err)
	}
	defer echoServer.Close()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, client, err := echoServer.ReadFromUDP(buf)
			if err != nil {
				return
			}
			_, _ = echoServer.WriteToUDP(buf[:n], client)
		}
	}()

	// 2. Setup UDP relay
	relay, err := NewUDPRelay()
	if err != nil {
		t.Fatalf("failed to create relay: %v", err)
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	go relay.Serve(stopCh)

	// 3. Send SOCKS5 UDP packet to relay
	clientConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Fatalf("failed to listen client UDP: %v", err)
	}
	defer clientConn.Close()

	// Header: [RSV 2B][FRAG 0][ATYP 1B (0x01)][IPv4 4B][Port 2B][Payload]
	targetIP := echoServer.LocalAddr().(*net.UDPAddr).IP.To4()
	targetPort := uint16(echoServer.LocalAddr().(*net.UDPAddr).Port)

	req := make([]byte, 10+5)
	req[0] = 0x00
	req[1] = 0x00
	req[2] = 0x00 // Fragment 0
	req[3] = 0x01 // IPv4
	copy(req[4:8], targetIP)
	binary.BigEndian.PutUint16(req[8:10], targetPort)
	copy(req[10:], []byte("hello"))

	_, err = clientConn.WriteToUDP(req, relay.LocalAddr())
	if err != nil {
		t.Fatalf("failed to send to relay: %v", err)
	}

	_ = clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	resp := make([]byte, 1024)
	n, _, err := clientConn.ReadFromUDP(resp)
	if err != nil {
		t.Fatalf("failed to read response from relay: %v", err)
	}

	if n < 10 {
		t.Fatalf("response too short: %d", n)
	}
	payload := string(resp[10:n])
	if payload != "hello" {
		t.Errorf("expected echo 'hello', got '%s'", payload)
	}
}
