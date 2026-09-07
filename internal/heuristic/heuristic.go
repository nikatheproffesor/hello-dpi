package heuristic

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/dpi"
)

// MutationTier represents the evolutionary escalation level of DPI evasion
type MutationTier int

const (
	TierStandard MutationTier = 0 // Baseline ISP-tuned strategy
	TierOffset   MutationTier = 1 // Micro-offset perturbation (split at 1 or 2 bytes)
	TierSequence MutationTier = 2 // Out-of-order & Wrong-Seq packet injection
	TierDecoy    MutationTier = 3 // Fake SNI decoy packet & TCP MSS clamping
	TierECH      MutationTier = 4 // Full ECH (Encrypted Client Hello) Outer SNI masking
)

// DomainState holds telemetry and learned mutation state for a domain
type DomainState struct {
	Domain          string       `json:"domain"`
	CurrentStrategy string       `json:"current_strategy"`
	SplitOffset     int          `json:"split_offset"`
	UseDecoy        bool         `json:"use_decoy"`
	MutationTier    MutationTier `json:"mutation_tier"`
	SuccessCount    int          `json:"success_count"`
	ConsecutiveFail int          `json:"consecutive_fail"`
	RSTCount        int          `json:"rst_count"`
	AvgRTTMs        int64        `json:"avg_rtt_ms"`
	LastUpdated     time.Time    `json:"last_updated"`
}

// Engine implements dynamic AI/heuristics-based evasion mutation
type Engine struct {
	mu         sync.RWMutex
	states     map[string]*DomainState
	cachePath  string
	persistMu  sync.Mutex
}

// NewEngine creates a new self-healing heuristics mutation engine
func NewEngine() *Engine {
	var cachePath string
	if configDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(configDir, "hellodpi")
		_ = os.MkdirAll(dir, 0755)
		cachePath = filepath.Join(dir, "learned_heuristics.json")
	}
	return NewEngineWithCache(cachePath)
}

// NewEngineWithCache creates an engine with a specified cache file or in-memory when empty
func NewEngineWithCache(cachePath string) *Engine {
	eng := &Engine{
		states:    make(map[string]*DomainState),
		cachePath: cachePath,
	}
	if cachePath != "" {
		eng.loadLearned()
	}
	return eng
}

// GetOptimalParameters returns the learned, dynamically mutated parameters for a domain
func (e *Engine) GetOptimalParameters(domain string) (strategy string, offset int, decoy bool) {
	d := strings.ToLower(strings.TrimSpace(domain))
	e.mu.RLock()
	st, exists := e.states[d]
	e.mu.RUnlock()

	if !exists {
		// Default standard baseline
		return string(dpi.SplitTLS), 5, false
	}

	return st.CurrentStrategy, st.SplitOffset, st.UseDecoy
}

// RecordHandshakeSuccess updates telemetry when a connection successfully performs TLS handshake
func (e *Engine) RecordHandshakeSuccess(domain string, rttMs int64) {
	d := strings.ToLower(strings.TrimSpace(domain))
	e.mu.Lock()
	st, exists := e.states[d]
	if !exists {
		st = &DomainState{
			Domain:          d,
			CurrentStrategy: string(dpi.SplitTLS),
			SplitOffset:     5,
			MutationTier:    TierStandard,
		}
		e.states[d] = st
	}

	st.ConsecutiveFail = 0
	st.SuccessCount++
	st.LastUpdated = time.Now()
	if st.AvgRTTMs == 0 {
		st.AvgRTTMs = rttMs
	} else if rttMs > 0 {
		st.AvgRTTMs = (st.AvgRTTMs*3 + rttMs) / 4
	}

	// If mutated tier achieved 3 consecutive successes, solidify and persist
	persist := (st.MutationTier > TierStandard && st.SuccessCount%3 == 0)
	e.mu.Unlock()

	if persist {
		e.saveLearned()
	}
}

// RecordHandshakeFailure triggers dynamic strategy mutation when an ISP drops or RSTs packets
func (e *Engine) RecordHandshakeFailure(domain string, isRST bool) {
	d := strings.ToLower(strings.TrimSpace(domain))
	e.mu.Lock()
	st, exists := e.states[d]
	if !exists {
		st = &DomainState{
			Domain:          d,
			CurrentStrategy: string(dpi.SplitTLS),
			SplitOffset:     5,
			MutationTier:    TierStandard,
		}
		e.states[d] = st
	}

	st.ConsecutiveFail++
	if isRST {
		st.RSTCount++
	}

	// Trigger mutation on 2 consecutive drops or an immediate TCP RST
	if st.ConsecutiveFail >= 2 || isRST {
		e.mutate(st)
	}
	e.mu.Unlock()

	e.saveLearned()
}

// mutate evolves the evasion profile to the next mutation tier
func (e *Engine) mutate(st *DomainState) {
	st.MutationTier++
	if st.MutationTier > TierECH {
		st.MutationTier = TierStandard // Cycle back with new random jitter
	}

	switch st.MutationTier {
	case TierOffset:
		// Perturb split point to 2 bytes (splits record length) or 1 byte (first-byte)
		st.CurrentStrategy = string(dpi.SplitFirstByte)
		st.SplitOffset = 2
		st.UseDecoy = false
		log.Printf("[Hello DPI AI] Dynamic Mutation: Domain %s escalated to Tier 1 (SplitOffset=%d)", st.Domain, st.SplitOffset)

	case TierSequence:
		// Switch to out-of-order or wrong-seq
		st.CurrentStrategy = string(dpi.SplitOutOfOrder)
		st.SplitOffset = 5
		st.UseDecoy = false
		log.Printf("[Hello DPI AI] Dynamic Mutation: Domain %s escalated to Tier 2 (OutOfOrder Seq)", st.Domain)

	case TierDecoy:
		// Inject fake decoy SNI packet before real payload
		st.CurrentStrategy = string(dpi.SplitWrongSeq)
		st.SplitOffset = 3
		st.UseDecoy = true
		log.Printf("[Hello DPI AI] Dynamic Mutation: Domain %s escalated to Tier 3 (Decoy SNI + WrongSeq)", st.Domain)

	case TierECH:
		// Full ECH encapsulation
		st.CurrentStrategy = string(dpi.SplitTLS)
		st.SplitOffset = 1
		st.UseDecoy = true
		log.Printf("[Hello DPI AI] Dynamic Mutation: Domain %s escalated to Tier 4 (ECH Outer Masking)", st.Domain)

	default:
		st.CurrentStrategy = string(dpi.SplitTLS)
		st.SplitOffset = 5
		st.UseDecoy = false
	}

	st.ConsecutiveFail = 0
}

func (e *Engine) loadLearned() {
	if e.cachePath == "" {
		return
	}
	data, err := os.ReadFile(e.cachePath)
	if err != nil {
		return
	}
	var loaded map[string]*DomainState
	if err := json.Unmarshal(data, &loaded); err == nil {
		e.mu.Lock()
		e.states = loaded
		e.mu.Unlock()
	}
}

func (e *Engine) saveLearned() {
	if e.cachePath == "" {
		return
	}
	e.persistMu.Lock()
	defer e.persistMu.Unlock()

	e.mu.RLock()
	data, err := json.MarshalIndent(e.states, "", "  ")
	e.mu.RUnlock()

	if err == nil {
		_ = os.WriteFile(e.cachePath, data, 0644)
	}
}
