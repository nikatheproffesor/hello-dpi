package engine

import (
	"testing"

	"github.com/hellodpi/hellodpi/internal/probe"
)

func BenchmarkGroupStrategyDispatch(b *testing.B) {
	ft := NewFallbackTracker(nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ft.GetActiveStrategy(probe.GroupDiscord)
	}
}

func BenchmarkClassifyDomain(b *testing.B) {
	domains := []string{
		"discord.com",
		"roblox.com",
		"youtube.com",
		"turkiye.gov.tr",
		"ziraatbank.com.tr",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := domains[i%len(domains)]
		_ = probe.ClassifyDomain(d)
	}
}
