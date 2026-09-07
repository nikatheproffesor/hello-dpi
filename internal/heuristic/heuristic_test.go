package heuristic

import (
	"testing"

	"github.com/hellodpi/hellodpi/internal/dpi"
)

func TestDynamicMutationProgression(t *testing.T) {
	eng := NewEngineWithCache("")
	domain := "test-censored-service.com"

	// Baseline check
	strat, offset, decoy := eng.GetOptimalParameters(domain)
	if strat != string(dpi.SplitTLS) || offset != 5 || decoy != false {
		t.Errorf("unexpected baseline parameters: %s, %d, %v", strat, offset, decoy)
	}

	// 1st failure (e.g. timeout) - does not mutate yet (needs 2 failures)
	eng.RecordHandshakeFailure(domain, false)
	strat, _, _ = eng.GetOptimalParameters(domain)
	if strat != string(dpi.SplitTLS) {
		t.Errorf("expected still baseline on 1st timeout, got %s", strat)
	}

	// 2nd failure - triggers Tier 1 mutation
	eng.RecordHandshakeFailure(domain, false)
	strat, offset, _ = eng.GetOptimalParameters(domain)
	if strat != string(dpi.SplitFirstByte) || offset != 2 {
		t.Errorf("expected Tier 1 mutation (first-byte, offset 2), got %s, %d", strat, offset)
	}

	// Immediate TCP RST - triggers Tier 2 mutation
	eng.RecordHandshakeFailure(domain, true)
	strat, _, _ = eng.GetOptimalParameters(domain)
	if strat != string(dpi.SplitOutOfOrder) {
		t.Errorf("expected Tier 2 mutation (out_of_order), got %s", strat)
	}

	// Success reinforces and locks in strategy
	eng.RecordHandshakeSuccess(domain, 25)
	stratAfterSuccess, _, _ := eng.GetOptimalParameters(domain)
	if stratAfterSuccess != string(dpi.SplitOutOfOrder) {
		t.Errorf("expected strategy to remain out_of_order after success, got %s", stratAfterSuccess)
	}
}
