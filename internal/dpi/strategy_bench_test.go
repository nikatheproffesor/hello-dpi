package dpi

import (
	"io"
	"net"
	"testing"
)

type discardConn struct {
	net.Conn
}

func (d *discardConn) Read(b []byte) (int, error)  { return 0, io.EOF }
func (d *discardConn) Write(b []byte) (int, error) { return len(b), nil }
func (d *discardConn) Close() error                { return nil }

func BenchmarkTLSRecordSplit(b *testing.B) {
	strat := NewTLSRecordSplitStrategy(5, 0)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	conn := &discardConn{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strat.Apply(conn, raw, info)
	}
}

func BenchmarkSNIMidSplit(b *testing.B) {
	strat := NewSNIMidSplitStrategy(0)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	conn := &discardConn{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strat.Apply(conn, raw, info)
	}
}

func BenchmarkWrongSeqAck(b *testing.B) {
	strat := NewWrongSeqAckStrategy(3, 0)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	conn := &discardConn{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strat.Apply(conn, raw, info)
	}
}

func BenchmarkWrongChecksum(b *testing.B) {
	strat := NewWrongChecksumStrategy(3, 0)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	conn := &discardConn{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strat.Apply(conn, raw, info)
	}
}

func BenchmarkOutOfOrder(b *testing.B) {
	strat := NewOutOfOrderStrategy(5, 0)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	conn := &discardConn{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strat.Apply(conn, raw, info)
	}
}

func BenchmarkTCPWindowMSS(b *testing.B) {
	strat := NewTCPWindowMSSStrategy(64, 0)
	raw := makeMockClientHello("discord.com")
	info := ParsePacket(raw)
	conn := &discardConn{}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = strat.Apply(conn, raw, info)
	}
}
