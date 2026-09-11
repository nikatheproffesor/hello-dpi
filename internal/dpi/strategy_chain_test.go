package dpi

import (
	"bytes"
	"net"
	"testing"
)

type dummyConn struct {
	net.Conn
	buf *bytes.Buffer
}

func (d *dummyConn) Write(b []byte) (int, error) {
	return d.buf.Write(b)
}

func (d *dummyConn) Close() error {
	return nil
}

func TestChainedStrategy(t *testing.T) {
	decoy := NewFakePacketStrategy(24, 1)
	sniMid := NewSNIMidSplitStrategy(1)

	chain := NewChainedStrategy("chain:decoy+sni", decoy, sniMid)
	if chain.Name() != "chain:decoy+sni" {
		t.Fatalf("unexpected chain name: %s", chain.Name())
	}

	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)

	buf := &bytes.Buffer{}
	conn := &dummyConn{buf: buf}

	err := chain.Apply(conn, raw, info)
	if err != nil {
		t.Fatalf("chain apply error: %v", err)
	}

	// buf must contain the 24-byte decoy followed by the exact intact raw ClientHello (split across writes)
	if buf.Len() != 24+len(raw) {
		t.Fatalf("expected written len %d (24-byte decoy + %d raw), got %d", 24+len(raw), len(raw), buf.Len())
	}

	// Verify that the payload after decoy is byte-for-byte equal to raw ClientHello
	writtenPayload := buf.Bytes()[24:]
	if !bytes.Equal(writtenPayload, raw) {
		t.Fatalf("payload was corrupted or duplicated during chain execution")
	}
}

func TestChainedStrategyRegistry(t *testing.T) {
	strat, ok := GetStrategy("chain:decoy+sni-mid")
	if !ok {
		t.Fatalf("expected chain:decoy+sni-mid to be registered")
	}
	if strat.Name() != "chain:decoy+sni-mid" {
		t.Fatalf("unexpected name: %s", strat.Name())
	}
}
