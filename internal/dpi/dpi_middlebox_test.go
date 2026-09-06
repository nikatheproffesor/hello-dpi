package dpi

import (
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockDPIMiddlebox simulates a stateful DPI firewall appliance (e.g. Sandvine / Huawei)
// which drops or resets TCP connections if the initial TCP packet contains a forbidden SNI keyword.
type mockDPIMiddlebox struct {
	listener net.Listener
	blocked  string
	addr     string
	closed   bool
	mu       sync.Mutex
}

func startMockDPIMiddlebox(t *testing.T, blockedSNI string) *mockDPIMiddlebox {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start mock DPI middlebox: %v", err)
	}

	m := &mockDPIMiddlebox{
		listener: l,
		blocked:  blockedSNI,
		addr:     l.Addr().String(),
	}

	go m.serve()
	return m
}

func (m *mockDPIMiddlebox) close() {
	m.mu.Lock()
	m.closed = true
	if m.listener != nil {
		_ = m.listener.Close()
	}
	m.mu.Unlock()
}

func (m *mockDPIMiddlebox) serve() {
	for {
		clientConn, err := m.listener.Accept()
		if err != nil {
			m.mu.Lock()
			closed := m.closed
			m.mu.Unlock()
			if closed {
				return
			}
			continue
		}

		go m.handleConnection(clientConn)
	}
}

func (m *mockDPIMiddlebox) handleConnection(client net.Conn) {
	defer client.Close()

	_ = client.SetDeadline(time.Now().Add(2 * time.Second))

	// Read the FIRST packet segment arriving from the client
	initialBuf := make([]byte, 4096)
	n, err := client.Read(initialBuf)
	if err != nil || n == 0 {
		return
	}
	firstPacket := initialBuf[:n]

	// DPI Inspection Rule:
	// If the entire blocked SNI keyword appears in the first packet (unfragmented),
	// the DPI middlebox identifies the target and immediately drops / resets the connection!
	if strings.Contains(string(firstPacket), m.blocked) {
		// Connection RST / Drop
		return
	}

	// Otherwise, if DPI is evaded (fragmented, decoy, or split), the middlebox allows the session
	// and returns a mock TLS ServerHello handshake response (0x16 0x03 0x03)
	mockServerHello := []byte{
		0x16, 0x03, 0x03, 0x00, 0x04,
		0x02, 0x00, 0x00, 0x00,
	}
	_, _ = client.Write(mockServerHello)

	// Keep connection open and drain remaining chunks so client's subsequent writes do not get broken pipe
	drainBuf := make([]byte, 4096)
	for {
		_ = client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, err := client.Read(drainBuf)
		if err != nil {
			break
		}
	}
}

func TestMockDPIMiddlebox_BlockedWithoutEvasion(t *testing.T) {
	dpiBox := startMockDPIMiddlebox(t, "discord.com")
	defer dpiBox.close()

	// Plain, unfragmented ClientHello containing "discord.com"
	rawClientHello := makeMockClientHello("discord.com")

	conn, err := net.Dial("tcp", dpiBox.addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	// Send without fragmentation
	_, _ = conn.Write(rawClientHello)

	reply := make([]byte, 5)
	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	_, rErr := conn.Read(reply)

	// Middlebox should have dropped the connection (EOF or timeout)
	if rErr == nil && reply[0] == 0x16 {
		t.Fatalf("Expected DPI middlebox to block plain unfragmented ClientHello!")
	}
}

func TestMockDPIMiddlebox_EvasionSuccess(t *testing.T) {
	dpiBox := startMockDPIMiddlebox(t, "discord.com")
	defer dpiBox.close()

	rawClientHello := makeMockClientHello("discord.com")
	info := ParsePacket(rawClientHello)

	strategiesToTest := []BypassStrategy{
		NewTLSRecordSplitStrategy(5, 2),
		NewSNIMidSplitStrategy(2),
		NewFakePacketStrategy(16, 2),
		NewReverseFragStrategy(3, 2),
		NewChunkedSplitStrategy(32, 2),
	}

	for _, strat := range strategiesToTest {
		t.Run("Strategy_"+strat.Name(), func(t *testing.T) {
			conn, err := net.DialTimeout("tcp", dpiBox.addr, 2*time.Second)
			if err != nil {
				t.Fatalf("Dial failed: %v", err)
			}
			defer conn.Close()

			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

			// Apply bypass strategy
			err = strat.Apply(conn, rawClientHello, info)
			if err != nil {
				t.Fatalf("Apply strategy failed: %v", err)
			}

			// Read response from middlebox
			reply := make([]byte, 5)
			n, rErr := io.ReadFull(conn, reply)
			if rErr != nil || n < 5 {
				t.Fatalf("Expected successful TLS ServerHello through DPI middlebox, got err: %v", rErr)
			}

			if reply[0] != 0x16 {
				t.Errorf("Expected TLS ServerHello byte 0x16, got %02x", reply[0])
			}
		})
	}
}

func TestRegression_KnownISPProfiles(t *testing.T) {
	profiles := []struct {
		name        string
		blockedHost string
		strategy    string
	}{
		{"Turkcell Superonline", "discord.com", string(SplitTLS)},
		{"TTNET Fiber", "roblox.com", string(SplitSNI)},
		{"Vodafone Mobile", "youtube.com", string(SplitDecoy)},
	}

	for _, p := range profiles {
		t.Run(p.name, func(t *testing.T) {
			dpiBox := startMockDPIMiddlebox(t, p.blockedHost)
			defer dpiBox.close()

			ch := makeMockClientHello(p.blockedHost)
			info := ParsePacket(ch)

			strat, ok := GetStrategy(p.strategy)
			if !ok {
				t.Fatalf("Strategy %s not registered", p.strategy)
			}

			conn, err := net.Dial("tcp", dpiBox.addr)
			if err != nil {
				t.Fatalf("Dial failed: %v", err)
			}
			defer conn.Close()

			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			if err := strat.Apply(conn, ch, info); err != nil {
				t.Fatalf("Strategy application failed: %v", err)
			}

			buf := make([]byte, 5)
			_, err = io.ReadFull(conn, buf)
			if err != nil || buf[0] != 0x16 {
				t.Errorf("ISP Profile %s failed to bypass DPI middlebox", p.name)
			}
		})
	}
}

// Deep Stateful Middlebox: Simulates inspection on TLS Record parsing layer
func TestStatefulDPIMiddlebox_RecordLayerInspection(t *testing.T) {
	// A stateful DPI engine that parses the TLS record layer:
	// If a single TLS record contains the full ClientHello with SNI "discord.com", it blocks!
	// If the ClientHello is divided across multiple records, Record 1 does not contain SNI,
	// and Record 2 does not have a Handshake header, causing the record-layer parser to bypass!
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read first TLS record header
		hdr := make([]byte, 5)
		if _, err := io.ReadFull(conn, hdr); err != nil {
			return
		}
		recLen := binary.BigEndian.Uint16(hdr[3:5])
		payload := make([]byte, recLen)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}

		// Check if record 1 is a complete ClientHello containing the SNI
		fullRecord := append(hdr, payload...)
		if strings.Contains(string(fullRecord), "discord.com") {
			// Middlebox caught single-record ClientHello -> DROP
			return
		}

		// Allowed! Send ServerHello
		_, _ = conn.Write([]byte{0x16, 0x03, 0x03, 0x00, 0x04, 0x02, 0x00, 0x00, 0x00})
		// Drain
		buf := make([]byte, 2048)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			if _, err := conn.Read(buf); err != nil {
				break
			}
		}
	}()

	// 1. Send with TLSRecordSplitStrategy
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	strat := NewTLSRecordSplitStrategy(5, 2)

	client, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer client.Close()

	if err := strat.Apply(client, raw, info); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	reply := make([]byte, 5)
	_ = client.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("Expected ServerHello through record-layer DPI, got: %v", err)
	}
	if reply[0] != 0x16 {
		t.Errorf("Expected 0x16, got %x", reply[0])
	}
}
