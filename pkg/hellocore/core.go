package hellocore

import (
	"context"
	"fmt"
	"sync"

	"github.com/hellodpi/hellodpi/internal/dns"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/voice"
)

// Engine is the unified, headless Hello DPI core engine supporting Desktop, Android, and iOS.
type Engine struct {
	mu          sync.Mutex
	server      *proxy.Server
	probeEngine *probe.Engine
	voiceOpt    *voice.Optimizer
	adapter     PlatformAdapter
	running     bool
	listenAddr  string
}

// MobileEngine is an alias for Engine to preserve 100% backward compatibility for mobile bindings.
type MobileEngine = Engine

// Config holds configuration parameters for the headless core engine
type Config struct {
	ListenAddr      string
	SplitMode       string // "auto", "tlsrec", "sni", "decoy", "reverse-frag", "first-byte", "chunked"
	DelayMs         int
	DoHEndpoint     string
	PlatformAdapter PlatformAdapter
}

// NewEngine creates a new headless Hello DPI instance
func NewEngine(cfg Config) *Engine {
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}
	if cfg.DelayMs <= 0 {
		cfg.DelayMs = 5
	}
	if cfg.DoHEndpoint == "" {
		cfg.DoHEndpoint = string(dns.Cloudflare)
	}

	probeEngine := probe.NewEngine()
	splitMode := dpi.SplitMode(cfg.SplitMode)
	delayMs := cfg.DelayMs

	if splitMode == "" || splitMode == dpi.SplitAuto {
		if tuned := probeEngine.GetLastResult(); tuned != nil && tuned.BestStrategy != "" {
			splitMode = dpi.SplitMode(tuned.BestStrategy)
			if tuned.BestDelayMs > 0 {
				delayMs = tuned.BestDelayMs
			}
		}
	}

	pCfg := proxy.Config{
		Addr:        cfg.ListenAddr,
		SplitMode:   splitMode,
		DelayMs:     delayMs,
		DoHEndpoint: cfg.DoHEndpoint,
		EnableDoH:   true,
	}

	srv := proxy.NewServer(pCfg)
	if tuned := probeEngine.GetLastResult(); tuned != nil && len(tuned.GroupStrategies) > 0 {
		srv.Orchestrator.UpdateGroupStrategies(tuned.GroupStrategies, tuned.GroupFallbacks)
	}

	return &Engine{
		server:      srv,
		probeEngine: probeEngine,
		voiceOpt:    voice.NewOptimizer(),
		adapter:     cfg.PlatformAdapter,
		listenAddr:  cfg.ListenAddr,
	}
}

// Start begins serving the proxy in background and activates the platform adapter
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return nil
	}

	if err := e.server.Listen(); err != nil {
		return fmt.Errorf("failed to bind proxy server: %w", err)
	}

	go func() {
		_ = e.server.Serve()
	}()

	if e.adapter != nil {
		if err := e.adapter.OnStart(); err != nil {
			_ = e.server.Close()
			return fmt.Errorf("platform adapter start failed: %w", err)
		}
	}

	e.running = true
	return nil
}

// Stop gracefully stops the proxy and deactivates the platform adapter
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	var adapterErr error
	if e.adapter != nil {
		adapterErr = e.adapter.OnStop()
	}

	err := e.server.Close()
	e.running = false
	if adapterErr != nil {
		return adapterErr
	}
	return err
}

// IsRunning returns whether the core proxy is actively listening
func (e *Engine) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.running
}

// AutoTune runs the live DPI probe and updates engine settings
func (e *Engine) AutoTune() (bestMode string, splitPos int, delayMs int, latencyMs int64, err error) {
	res := e.probeEngine.RunProbe()
	if res.BypassVerified {
		e.server.UpdateEngineConfig(dpi.SplitMode(res.BestStrategy), res.BestSplitPos, res.BestDelayMs)
		return res.BestStrategy, res.BestSplitPos, res.BestDelayMs, res.BestLatencyMs, nil
	}
	return res.BestStrategy, res.BestSplitPos, res.BestDelayMs, res.BestLatencyMs, fmt.Errorf("fallback mode applied")
}

// SyncRules fetches remote rule updates
func (e *Engine) SyncRules(url string) error {
	return e.server.Rules.SyncRemote(url)
}

// RulesStats returns rule statistics
func (e *Engine) RulesStats() (version string, directCount int, interceptCount int) {
	v, d, i, _ := e.server.Rules.Stats()
	return v, d, i
}

// EvaluateHost determines if a host should bypass DPI
func (e *Engine) EvaluateHost(host string) int {
	return int(e.server.Rules.Evaluate(host))
}

// TestVoiceConnectivity verifies WebRTC voice channel connectivity
func (e *Engine) TestVoiceConnectivity() (responding bool, latencyMs int64) {
	st := e.voiceOpt.TestVoiceConnectivity()
	return st.STUNResponding, st.UDPLatencyMs
}

// ResolveHost resolves a hostname using the internal DoH resolver
func (e *Engine) ResolveHost(host string) (string, error) {
	return e.server.Resolver.Resolve(context.Background(), host)
}

var (
	defaultEngine   *Engine
	defaultEngineMu sync.Mutex
)

// StartMobileDefault starts the global Hello DPI mobile core proxy on 127.0.0.1:port
func StartMobileDefault(port int) error {
	defaultEngineMu.Lock()
	defer defaultEngineMu.Unlock()

	if defaultEngine != nil && defaultEngine.IsRunning() {
		return nil
	}
	if port <= 0 {
		port = 8080
	}
	defaultEngine = NewEngine(Config{
		ListenAddr:      fmt.Sprintf("127.0.0.1:%d", port),
		SplitMode:       "auto",
		DelayMs:         5,
		PlatformAdapter: NewAndroidVPNAdapter(),
	})
	return defaultEngine.Start()
}

// StopMobileDefault stops the active mobile engine
func StopMobileDefault() error {
	defaultEngineMu.Lock()
	defer defaultEngineMu.Unlock()

	if defaultEngine != nil {
		err := defaultEngine.Stop()
		defaultEngine = nil
		return err
	}
	return nil
}

// IsMobileRunning returns true if the default mobile engine is active
func IsMobileRunning() bool {
	defaultEngineMu.Lock()
	defer defaultEngineMu.Unlock()

	return defaultEngine != nil && defaultEngine.IsRunning()
}
