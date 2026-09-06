package engine

import (
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"

	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/probe"
)

// GroupToIndex maps a DomainGroup to a zero-alloc contiguous index (0..3)
func GroupToIndex(g probe.DomainGroup) int {
	switch g {
	case probe.GroupDiscord:
		return 0
	case probe.GroupRoblox:
		return 1
	case probe.GroupWeb:
		return 2
	default:
		return 3
	}
}

// FallbackTracker manages runtime degradation detection and automatic failover
type FallbackTracker struct {
	mu             sync.RWMutex
	strategies     map[probe.DomainGroup][]dpi.BypassStrategy
	activeIndex    map[probe.DomainGroup]int
	failureStreaks map[probe.DomainGroup]int
	fastActive     [4]dpi.BypassStrategy
	totalFallbacks uint64
	onFallback     func(group probe.DomainGroup, fromStrat, toStrat string)
}

// NewFallbackTracker initializes a fallback tracker with defaults
func NewFallbackTracker(onFallback func(group probe.DomainGroup, fromStrat, toStrat string)) *FallbackTracker {
	ft := &FallbackTracker{
		strategies:     make(map[probe.DomainGroup][]dpi.BypassStrategy),
		activeIndex:    make(map[probe.DomainGroup]int),
		failureStreaks: make(map[probe.DomainGroup]int),
		onFallback:     onFallback,
	}

	// Initialize sensible default fallback chains
	ft.strategies[probe.GroupDiscord] = []dpi.BypassStrategy{
		dpi.ResolveStrategy(string(dpi.SplitOutOfOrder)),
		dpi.ResolveStrategy(string(dpi.SplitWrongSeq)),
		dpi.ResolveStrategy(string(dpi.SplitTLS)),
	}
	ft.strategies[probe.GroupRoblox] = []dpi.BypassStrategy{
		dpi.ResolveStrategy(string(dpi.SplitWrongSeq)),
		dpi.ResolveStrategy(string(dpi.SplitOutOfOrder)),
		dpi.ResolveStrategy(string(dpi.SplitTLS)),
	}
	ft.strategies[probe.GroupWeb] = []dpi.BypassStrategy{
		dpi.ResolveStrategy(string(dpi.SplitTLS)),
		dpi.ResolveStrategy(string(dpi.SplitSNI)),
		dpi.ResolveStrategy(string(dpi.SplitChunked)),
	}
	ft.strategies[probe.GroupSafe] = []dpi.BypassStrategy{
		nil, // Direct pass-through
	}

	ft.rebuildFastActiveLocked()
	return ft
}

func (ft *FallbackTracker) rebuildFastActiveLocked() {
	groups := []probe.DomainGroup{probe.GroupDiscord, probe.GroupRoblox, probe.GroupWeb, probe.GroupSafe}
	for _, g := range groups {
		idx := GroupToIndex(g)
		strats := ft.strategies[g]
		if len(strats) == 0 {
			if g == probe.GroupSafe {
				ft.fastActive[idx] = nil
			} else {
				ft.fastActive[idx] = dpi.DefaultStrategy()
			}
			continue
		}
		actIdx := ft.activeIndex[g]
		if actIdx >= len(strats) {
			actIdx = 0
		}
		ft.fastActive[idx] = strats[actIdx]
	}
}

// SetGroupChains updates the ranked fallback chains for all domain groups
func (ft *FallbackTracker) SetGroupChains(primary map[probe.DomainGroup]string, fallbacks map[probe.DomainGroup][]string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	for g, list := range fallbacks {
		var strats []dpi.BypassStrategy
		for _, name := range list {
			if s, ok := dpi.GetStrategy(name); ok {
				strats = append(strats, s)
			}
		}
		if len(strats) == 0 {
			// Fallback to primary or default
			if pName, ok := primary[g]; ok {
				strats = append(strats, dpi.ResolveStrategy(pName))
			} else {
				strats = append(strats, dpi.DefaultStrategy())
			}
		}
		ft.strategies[g] = strats
		ft.activeIndex[g] = 0
		ft.failureStreaks[g] = 0
	}

	ft.rebuildFastActiveLocked()
}

// SetGroupStrategies directly configures the strategy fallback slice for a specific group
func (ft *FallbackTracker) SetGroupStrategies(group probe.DomainGroup, strats []dpi.BypassStrategy) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	ft.strategies[group] = strats
	ft.activeIndex[group] = 0
	ft.failureStreaks[group] = 0
	ft.rebuildFastActiveLocked()
}

// GetActiveStrategy retrieves the currently active strategy for a given domain group (O(1) fast path)
func (ft *FallbackTracker) GetActiveStrategy(group probe.DomainGroup) dpi.BypassStrategy {
	ft.mu.RLock()
	idx := GroupToIndex(group)
	strat := ft.fastActive[idx]
	ft.mu.RUnlock()
	return strat
}

// RecordSuccess clears any failure streak on a successful bypass
func (ft *FallbackTracker) RecordSuccess(group probe.DomainGroup) {
	ft.mu.Lock()
	ft.failureStreaks[group] = 0
	ft.mu.Unlock()
}

// GetFailureStreak returns the current consecutive failure streak for a group
func (ft *FallbackTracker) GetFailureStreak(group probe.DomainGroup) int {
	ft.mu.RLock()
	defer ft.mu.RUnlock()
	return ft.failureStreaks[group]
}

// RecordFailure increments the failure counter and advances the active strategy if degraded
func (ft *FallbackTracker) RecordFailure(group probe.DomainGroup) (dpi.BypassStrategy, bool) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	strats := ft.strategies[group]
	if len(strats) <= 1 {
		return nil, false
	}

	ft.failureStreaks[group]++
	// Degrade after 2 consecutive failures
	if ft.failureStreaks[group] >= 2 {
		oldIdx := ft.activeIndex[group]
		newIdx := (oldIdx + 1) % len(strats)
		ft.activeIndex[group] = newIdx
		ft.failureStreaks[group] = 0
		atomic.AddUint64(&ft.totalFallbacks, 1)

		ft.rebuildFastActiveLocked()

		fromName := "none"
		if strats[oldIdx] != nil {
			fromName = strats[oldIdx].Name()
		}
		toName := "none"
		if strats[newIdx] != nil {
			toName = strats[newIdx].Name()
		}

		log.Printf("[Hello DPI Fallback] Domain group [%s] degraded. Switched from [%s] -> [%s]", group, fromName, toName)
		if ft.onFallback != nil {
			go ft.onFallback(group, fromName, toName)
		}
		return strats[newIdx], true
	}

	return nil, false
}

// ApplyWithFallback executes the group's active bypass strategy with instant in-band fallback
func (ft *FallbackTracker) ApplyWithFallback(group probe.DomainGroup, conn net.Conn, payload []byte, info dpi.ParsedInfo) error {
	if group == probe.GroupSafe {
		_, err := conn.Write(payload)
		return err
	}

	ft.mu.RLock()
	idx := GroupToIndex(group)
	primary := ft.fastActive[idx]
	strats := ft.strategies[group]
	activeIdx := ft.activeIndex[group]
	ft.mu.RUnlock()

	// 1. Try active primary strategy (O(1) lookup)
	if primary != nil {
		err := primary.Apply(conn, payload, info)
		if err == nil {
			ft.RecordSuccess(group)
			return nil
		}
		log.Printf("[Hello DPI Engine] Strategy [%s] failed on group [%s]: %v. Triggering fallback...", primary.Name(), group, err)
	}

	// 2. Strategy failed: record failure and evaluate circuit breaker
	ft.RecordFailure(group)

	// 3. In-band failover: attempt next strategy in chain if connection supports re-write
	for i := 0; i < len(strats); i++ {
		nextIdx := (activeIdx + 1 + i) % len(strats)
		nextStrat := strats[nextIdx]
		if nextStrat != nil && nextStrat != primary {
			if err := nextStrat.Apply(conn, payload, info); err == nil {
				return nil
			}
		}
	}

	// 4. Ultimate Fail-Open: attempt raw write so client connection isn't abruptly severed
	_, err := conn.Write(payload)
	if err != nil {
		return fmt.Errorf("fail-open write failed: %w", err)
	}
	return nil
}

// TotalFallbacks returns the total count of automatic fallbacks triggered
func (ft *FallbackTracker) TotalFallbacks() uint64 {
	return atomic.LoadUint64(&ft.totalFallbacks)
}
