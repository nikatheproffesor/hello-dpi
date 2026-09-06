package dpi

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
)

// mockConn captures all bytes written to it
type mockConn struct {
	net.Conn
	buf    bytes.Buffer
	chunks [][]byte
}

func (m *mockConn) Write(b []byte) (int, error) {
	chunk := make([]byte, len(b))
	copy(chunk, b)
	m.chunks = append(m.chunks, chunk)
	return m.buf.Write(b)
}

func (m *mockConn) Read(b []byte) (int, error) {
	return 0, io.EOF
}

func (m *mockConn) Close() error {
	return nil
}

func makeMockClientHello(sni string) []byte {
	sniBytes := []byte(sni)
	sNameLen := len(sniBytes)

	// SNI Extension
	extSNILen := 5 + sNameLen
	extSNI := make([]byte, 4+extSNILen)
	binary.BigEndian.PutUint16(extSNI[0:2], 0x0000)
	binary.BigEndian.PutUint16(extSNI[2:4], uint16(extSNILen))
	binary.BigEndian.PutUint16(extSNI[4:6], uint16(3+sNameLen))
	extSNI[6] = 0x00
	binary.BigEndian.PutUint16(extSNI[7:9], uint16(sNameLen))
	copy(extSNI[9:], sniBytes)

	// Extensions
	extsBlock := make([]byte, 2+len(extSNI))
	binary.BigEndian.PutUint16(extsBlock[0:2], uint16(len(extSNI)))
	copy(extsBlock[2:], extSNI)

	// Body
	body := make([]byte, 2+32+1+4+2+len(extsBlock))
	body[0] = 0x03
	body[1] = 0x03
	body[34] = 0x00
	body[35] = 0x00
	body[36] = 0x02
	body[37] = 0x13
	body[38] = 0x01
	body[39] = 0x01
	body[40] = 0x00
	copy(body[41:], extsBlock)

	handshake := make([]byte, 4+len(body))
	handshake[0] = 0x01
	handshake[1] = byte((len(body) >> 16) & 0xFF)
	handshake[2] = byte((len(body) >> 8) & 0xFF)
	handshake[3] = byte(len(body) & 0xFF)
	copy(handshake[4:], body)

	record := make([]byte, 5+len(handshake))
	record[0] = 0x16
	record[1] = 0x03
	record[2] = 0x01
	binary.BigEndian.PutUint16(record[3:5], uint16(len(handshake)))
	copy(record[5:], handshake)

	return record
}

func TestStrategyRegistry(t *testing.T) {
	strats := AllStrategies()
	if len(strats) < 6 {
		t.Fatalf("Expected at least 6 registered strategies, got %d", len(strats))
	}

	names := []string{
		string(SplitTLS),
		string(SplitSNI),
		string(SplitDecoy),
		string(SplitFirstByte),
		string(SplitChunked),
		string(SplitHTTPMix),
		string(SplitAuto),
		string(SplitReverseFrag),
	}

	for _, name := range names {
		s, ok := GetStrategy(name)
		if !ok || s == nil {
			t.Errorf("Expected strategy '%s' to be registered", name)
		}
	}
}

func TestTLSRecordSplitStrategy(t *testing.T) {
	strat := NewTLSRecordSplitStrategy(5, 1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(mc.chunks) < 2 {
		t.Fatalf("Expected at least 2 chunks, got %d", len(mc.chunks))
	}

	// Verify both chunks have valid 0x16 TLS header
	for i, c := range mc.chunks {
		if c[0] != 0x16 {
			t.Errorf("Chunk %d missing TLS record byte 0x16", i)
		}
	}
}

func TestSNIMidSplitStrategy(t *testing.T) {
	strat := NewSNIMidSplitStrategy(1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	if info.Host != "discord.com" {
		t.Fatalf("Expected SNI host 'discord.com', got '%s'", info.Host)
	}

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(mc.chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(mc.chunks))
	}

	// Reassembled stream must match original
	reassembled := append(mc.chunks[0], mc.chunks[1]...)
	if !bytes.Equal(reassembled, raw) {
		t.Fatalf("Reassembled stream does not match raw ClientHello")
	}

	// Chunk 1 should not contain "discord.com" in full
	if bytes.Contains(mc.chunks[0], []byte("discord.com")) {
		t.Errorf("Chunk 1 contains full SNI string 'discord.com'")
	}
	if bytes.Contains(mc.chunks[1], []byte("discord.com")) {
		t.Errorf("Chunk 2 contains full SNI string 'discord.com'")
	}
}

func TestFakePacketStrategy(t *testing.T) {
	strat := NewFakePacketStrategy(16, 1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Chunk 0 must be decoy
	if len(mc.chunks) < 3 {
		t.Fatalf("Expected decoy + 2 payload chunks, got %d chunks", len(mc.chunks))
	}
	if mc.chunks[0][0] != 0x17 && mc.chunks[0][0] != 0x15 {
		t.Errorf("Decoy packet unexpected record type %x", mc.chunks[0][0])
	}
}

func TestHTTPHostTrickStrategy(t *testing.T) {
	strat := NewHTTPHostTrickStrategy(TrickHostCaseMix, 1)
	httpReq := []byte("GET / HTTP/1.1\r\nHost: example.com\r\nAccept: */*\r\n\r\n")
	info := ParsePacket(httpReq)

	if info.Type != TypeHTTPRequest {
		t.Fatalf("Expected TypeHTTPRequest, got %d", info.Type)
	}

	mc := &mockConn{}
	err := strat.Apply(mc, httpReq, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(mc.chunks) != 2 {
		t.Fatalf("Expected 2 chunks for HTTP Host trick, got %d", len(mc.chunks))
	}

	full := mc.buf.Bytes()
	// Must contain modified Host casing "hoSt"
	if !bytes.Contains(full, []byte("hoSt:")) {
		t.Errorf("Expected 'hoSt:' in output, got %s", string(full))
	}
}

func TestChunkedSplitStrategy(t *testing.T) {
	strat := NewChunkedSplitStrategy(16, 1)
	data := []byte("0123456789012345678901234567890123456789") // 40 bytes
	info := ParsedInfo{Type: TypeUnknown}

	mc := &mockConn{}
	err := strat.Apply(mc, data, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(mc.chunks) != 3 { // 16 + 16 + 8 = 3 chunks
		t.Fatalf("Expected 3 chunks, got %d", len(mc.chunks))
	}
	if !bytes.Equal(mc.buf.Bytes(), data) {
		t.Fatalf("Data mismatch after chunked streaming")
	}
}

func TestFirstByteSplitStrategy(t *testing.T) {
	strat := NewFirstByteSplitStrategy(1)
	data := []byte("HELLO WORLD")
	info := ParsedInfo{Type: TypeUnknown}

	mc := &mockConn{}
	err := strat.Apply(mc, data, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(mc.chunks) != 2 {
		t.Fatalf("Expected 2 chunks, got %d", len(mc.chunks))
	}
	if len(mc.chunks[0]) != 1 || mc.chunks[0][0] != 'H' {
		t.Fatalf("First chunk should be 1 byte 'H', got %s", string(mc.chunks[0]))
	}
}

func TestAdaptiveStrategy(t *testing.T) {
	strat := NewAdaptiveStrategy()

	// 1. TLS
	rawTLS := makeMockClientHello("discord.com")
	infoTLS := ParsePacket(rawTLS)
	mcTLS := &mockConn{}
	if err := strat.Apply(mcTLS, rawTLS, infoTLS); err != nil {
		t.Fatalf("TLS adaptive failed: %v", err)
	}
	if len(mcTLS.chunks) < 2 {
		t.Fatalf("Expected multiple chunks for TLS, got %d", len(mcTLS.chunks))
	}

	// 2. HTTP
	rawHTTP := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	infoHTTP := ParsePacket(rawHTTP)
	mcHTTP := &mockConn{}
	if err := strat.Apply(mcHTTP, rawHTTP, infoHTTP); err != nil {
		t.Fatalf("HTTP adaptive failed: %v", err)
	}
	if !bytes.Contains(mcHTTP.buf.Bytes(), []byte("hoSt:")) {
		t.Errorf("Expected 'hoSt:' trick on adaptive HTTP")
	}

	// 3. Raw unknown
	rawUnknown := []byte("PING\r\n")
	infoUnknown := ParsePacket(rawUnknown)
	mcUnknown := &mockConn{}
	if err := strat.Apply(mcUnknown, rawUnknown, infoUnknown); err != nil {
		t.Fatalf("Unknown adaptive failed: %v", err)
	}
	if len(mcUnknown.chunks) != 2 {
		t.Fatalf("Expected 2 chunks for unknown first-byte split, got %d", len(mcUnknown.chunks))
	}
}

func TestWrongSeqAckStrategy(t *testing.T) {
	strat := NewWrongSeqAckStrategy(3, 1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(mc.chunks) < 3 {
		t.Fatalf("Expected decoy + record chunks, got %d", len(mc.chunks))
	}
	// First chunk should be TLS alert decoy (0x15)
	if mc.chunks[0][0] != 0x15 {
		t.Errorf("Expected 0x15 TLS alert decoy, got %x", mc.chunks[0][0])
	}
}

func TestWrongChecksumStrategy(t *testing.T) {
	strat := NewWrongChecksumStrategy(3, 1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(mc.chunks) < 3 {
		t.Fatalf("Expected wrong checksum decoy + chunks, got %d", len(mc.chunks))
	}
}

func TestTCPWindowMSSStrategy(t *testing.T) {
	strat := NewTCPWindowMSSStrategy(32, 1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(mc.chunks) < 2 {
		t.Fatalf("Expected multiple MSS chunks, got %d", len(mc.chunks))
	}
}

func TestQUICBlock(t *testing.T) {
	if !ShouldBlockQUIC("udp", "1.1.1.1:443") {
		t.Errorf("Expected UDP:443 to be blocked for QUIC")
	}
	if ShouldBlockQUIC("tcp", "1.1.1.1:443") {
		t.Errorf("TCP:443 should NOT be blocked for QUIC")
	}
	if ShouldBlockQUIC("udp", "1.1.1.1:80") {
		t.Errorf("UDP:80 should NOT be blocked for QUIC")
	}

	reply := SOCKS5QUICRejectReply()
	if reply[1] != 0x07 {
		t.Errorf("Expected command not supported 0x07, got %x", reply[1])
	}
}

func TestOutOfOrderStrategy(t *testing.T) {
	strat := NewOutOfOrderStrategy(5, 1)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	mc := &mockConn{}
	err := strat.Apply(mc, raw, info)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if len(mc.chunks) < 3 {
		t.Fatalf("Expected decoy + out-of-order chunks, got %d", len(mc.chunks))
	}
}

