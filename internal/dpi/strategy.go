package dpi

import (
	"fmt"
	"net"
	"sync"
)

// BypassStrategy defines the contract for DPI evasion techniques.
type BypassStrategy interface {
	Name() string
	Apply(conn net.Conn, data []byte, info ParsedInfo) error
}

var (
	strategiesMu sync.RWMutex
	strategies   = make(map[string]BypassStrategy)
	strategyList []BypassStrategy
)

// RegisterStrategy adds a strategy to the global registry.
func RegisterStrategy(strat BypassStrategy) {
	strategiesMu.Lock()
	defer strategiesMu.Unlock()

	name := strat.Name()
	if _, exists := strategies[name]; !exists {
		strategyList = append(strategyList, strat)
	}
	strategies[name] = strat
}

// GetStrategy retrieves a strategy by its registered name.
func GetStrategy(name string) (BypassStrategy, bool) {
	strategiesMu.RLock()
	defer strategiesMu.RUnlock()

	s, ok := strategies[name]
	return s, ok
}

// AllStrategies returns a slice of all registered strategies in registration order.
func AllStrategies() []BypassStrategy {
	strategiesMu.RLock()
	defer strategiesMu.RUnlock()

	result := make([]BypassStrategy, len(strategyList))
	copy(result, strategyList)
	return result
}

// DefaultStrategy returns the standard RFC TLS Record Splitting strategy.
func DefaultStrategy() BypassStrategy {
	if s, ok := GetStrategy(string(SplitTLS)); ok {
		return s
	}
	return NewTLSRecordSplitStrategy(5, 5)
}

// ResolveStrategy attempts to find a matching strategy or returns a fallback.
func ResolveStrategy(name string) BypassStrategy {
	if s, ok := GetStrategy(name); ok {
		return s
	}
	return DefaultStrategy()
}

// StrategyError captures an error produced while applying a bypass strategy.
type StrategyError struct {
	StrategyName string
	Err          error
}

func (e *StrategyError) Error() string {
	return fmt.Sprintf("strategy [%s] failed: %v", e.StrategyName, e.Err)
}

func (e *StrategyError) Unwrap() error {
	return e.Err
}
