package dpi

import (
	"fmt"
	"net"
	"strings"
)

// ChainedStrategy combines multiple BypassStrategies in sequential order.
// This matches and exceeds advanced multi-vector desync engines (like Zapret's chaining),
// enabling combinations such as Decoy Low-TTL injection followed by SNI-Mid splitting.
type ChainedStrategy struct {
	name       string
	strategies []BypassStrategy
}

// NewChainedStrategy creates a composite strategy from an ordered slice of strategies.
func NewChainedStrategy(name string, strats ...BypassStrategy) *ChainedStrategy {
	cleanName := name
	if cleanName == "" {
		names := make([]string, 0, len(strats))
		for _, s := range strats {
			names = append(names, s.Name())
		}
		cleanName = "chain:" + strings.Join(names, "+")
	}
	return &ChainedStrategy{
		name:       cleanName,
		strategies: strats,
	}
}

func (c *ChainedStrategy) Name() string {
	return c.name
}

func (c *ChainedStrategy) Strategies() []BypassStrategy {
	return c.strategies
}

func (c *ChainedStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	for _, strat := range c.strategies {
		if err := strat.Apply(conn, data, info); err != nil {
			return &StrategyError{
				StrategyName: strat.Name(),
				Err:          fmt.Errorf("chain step failed in [%s]: %w", c.name, err),
			}
		}
	}
	return nil
}

func init() {
	// Register popular high-resilience chains
	decoy := NewFakePacketStrategy(24, 2)
	sniMid := NewSNIMidSplitStrategy(2)
	tlsRec := NewTLSRecordSplitStrategy(5, 2)
	wrongSeq := NewWrongSeqAckStrategy(3, 2)

	// Chain 1: Decoy + SNI-Mid
	RegisterStrategy(NewChainedStrategy("chain:decoy+sni-mid", decoy, sniMid))
	// Chain 2: Wrong SEQ/ACK + TLS Record Split
	RegisterStrategy(NewChainedStrategy("chain:wrong-seq+tlsrec", wrongSeq, tlsRec))
}
