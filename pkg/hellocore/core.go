package hellocore

import (
	"context"
	"fmt"
	"sync"

	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/voice"
)

// MobileEngine is the core headless Hello DPI engine designed for mobile (Android/iOS) and embedded use
type MobileEngine struct {
	mu          sync.Mutex
	server      *proxy.Server
	probeEngine *probe.Engine
	voiceOpt    *voice.Optimizer
	running     bool
	listenAddr  string
}

// Config holds configuration parameters for the mobile engine
type Config struct {
	ListenAddr string
	SplitMode  string // "auto", "tlsrec", "first-byte", "chunked"
	DelayMs    int
	DoHEndpoint string
}

// NewEngine creates a new headless Hello DPI instance
func NewEngine(cfg Config) *MobileEngine {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}
	if cfg.DelayMs <= 0 {
		cfg.DelayMs = 5
	}
	if cfg.DoHEndpoint == "" {
		cfg.DoHEndpoint = string(doh.Cloudflare)
	}

	pCfg := proxy.Config{
		Addr:        cfg.ListenAddr,
		SplitMode:   dpi.SplitMode(cfg.SplitMode),
		DelayMs:     cfg.DelayMs,
		DoHEndpoint: cfg.DoHEndpoint,
		EnableDoH:   true,
	}

	return &MobileEngine{
		server:      proxy.NewServer(pCfg),
		probeEngine: probe.NewEngine(),
		voiceOpt:    voice.NewOptimizer(),
		listenAddr:  cfg.ListenAddr,
	}
}

// Start begins serving the proxy in background
func (m *MobileEngine) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	go func() {
		_ = m.server.Start()
	}()

	m.running = true
	return nil
}

// Stop gracefully stops the proxy
func (m *MobileEngine) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	err := m.server.Close()
	m.running = false
	return err
}

// IsRunning returns whether the core proxy is actively listening
func (m *MobileEngine) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// AutoTune runs the live DPI probe and updates engine settings
func (m *MobileEngine) AutoTune() (bestMode string, splitPos int, delayMs int, latencyMs int64, err error) {
	res := m.probeEngine.RunProbe()
	if res.BypassVerified {
		m.server.UpdateEngineConfig(dpi.SplitMode(res.BestMode), res.BestSplitPos, res.BestDelayMs)
		return res.BestMode, res.BestSplitPos, res.BestDelayMs, res.BestLatencyMs, nil
	}
	return res.BestMode, res.BestSplitPos, res.BestDelayMs, res.BestLatencyMs, fmt.Errorf("fallback mode applied")
}

// SyncRules fetches remote rule updates
func (m *MobileEngine) SyncRules(url string) error {
	return m.server.Rules.SyncRemote(url)
}

// RulesStats returns rule statistics
func (m *MobileEngine) RulesStats() (version string, directCount int, interceptCount int) {
	v, d, i, _ := m.server.Rules.Stats()
	return v, d, i
}

// EvaluateHost determines if a host should bypass DPI
func (m *MobileEngine) EvaluateHost(host string) int {
	return int(m.server.Rules.Evaluate(host))
}

// TestVoiceConnectivity verifies WebRTC voice channel connectivity
func (m *MobileEngine) TestVoiceConnectivity() (responding bool, latencyMs int64) {
	st := m.voiceOpt.TestVoiceConnectivity()
	return st.STUNResponding, st.UDPLatencyMs
}

// ResolveHost resolves a hostname using the internal DoH resolver
func (m *MobileEngine) ResolveHost(host string) (string, error) {
	return m.server.Resolver.Resolve(context.Background(), host)
}
