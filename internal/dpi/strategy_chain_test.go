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
	strat1 := NewFirstByteSplitStrategy(1)
	strat2 := NewSNIMidSplitStrategy(1)

	chain := NewChainedStrategy("chain:first-byte+sni", strat1, strat2)
	if chain.Name() != "chain:first-byte+sni" {
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

	if buf.Len() == 0 {
		t.Fatalf("expected data written to connection")
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
