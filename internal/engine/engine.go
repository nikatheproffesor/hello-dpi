package engine

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/divert"
	"github.com/hellodpi/hellodpi/internal/dns"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/rules"
	"github.com/hellodpi/hellodpi/internal/tunnel"
)

// Config holds settings for the core orchestration engine
type Config struct {
	StrategyName string
	SplitOffset  int
	DelayMs      int
	DoHEndpoint  string
	EnableDoH    bool
	DialTimeout  time.Duration
}

// Orchestrator coordinates L7 proxy routing, DNS resolution, domain-group DPI evasion,
// runtime fallback circuit breaking, and selective L3/L4 kernel divert integration.
type Orchestrator struct {
	mu           sync.RWMutex
	Rules        *rules.Engine
	Resolver     *dns.Resolver
	Strategy     dpi.BypassStrategy
	Fallback     *FallbackTracker
	SplitOffset  int
	DelayMs      int
	DialTimeout  time.Duration
	KernelActive bool
}

// NewOrchestrator creates a new unified orchestration engine
func NewOrchestrator(cfg Config) *Orchestrator {
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 10 * time.Second
	}
	if cfg.DelayMs <= 0 {
		cfg.DelayMs = 5
	}
	if cfg.SplitOffset <= 0 {
		cfg.SplitOffset = 5
	}

	strat := dpi.ResolveStrategy(cfg.StrategyName)

	return &Orchestrator{
		Rules:       rules.NewEngine(),
		Resolver:    dns.NewResolver(cfg.DoHEndpoint, cfg.EnableDoH),
		Strategy:    strat,
		Fallback:    NewFallbackTracker(nil),
		SplitOffset: cfg.SplitOffset,
		DelayMs:     cfg.DelayMs,
		DialTimeout: cfg.DialTimeout,
	}
}

// UpdateStrategy dynamically updates the active bypass strategy and parameters
func (o *Orchestrator) UpdateStrategy(name string, splitOffset int, delayMs int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.Strategy = dpi.ResolveStrategy(name)
	if splitOffset > 0 {
		o.SplitOffset = splitOffset
	}
	if delayMs > 0 {
		o.DelayMs = delayMs
	}
	o.Fallback.SetGroupChains(map[probe.DomainGroup]string{
		probe.GroupWeb:     name,
		probe.GroupDiscord: name,
	}, nil)

	log.Printf("[Hello DPI Engine] Active strategy updated: %s (offset=%d, delay=%dms)", name, o.SplitOffset, o.DelayMs)
}

// UpdateGroupStrategies updates the domain-group specific strategies and fallback chains from Auto-Tuning
func (o *Orchestrator) UpdateGroupStrategies(primary map[probe.DomainGroup]string, fallbacks map[probe.DomainGroup][]string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.Fallback.SetGroupChains(primary, fallbacks)
	if webStrat, ok := primary[probe.GroupWeb]; ok {
		o.Strategy = dpi.ResolveStrategy(webStrat)
	}
	log.Printf("[Hello DPI Engine] Group strategies updated: discord=%s, roblox=%s, web=%s",
		primary[probe.GroupDiscord], primary[probe.GroupRoblox], primary[probe.GroupWeb])
}

// HandleTunnel processes an established tunnel (from HTTP CONNECT or SOCKS5)
func (o *Orchestrator) HandleTunnel(clientConn net.Conn, reader *bufio.Reader, targetHost, targetPort string) error {
	destAddr, action, err := o.resolveDestination(targetHost, targetPort)
	if err != nil {
		return fmt.Errorf("failed to resolve target %s: %w", targetHost, err)
	}

	group := probe.ClassifyDomain(targetHost)

	// If rule requires kernel divert (e.g. Roblox direct socket), activate silently in background
	if action == rules.ActionKernel {
		go o.ensureKernelRunning(targetHost)
	}

	// Dial target server
	targetConn, err := net.DialTimeout("tcp", destAddr, o.DialTimeout)
	if err != nil {
		return fmt.Errorf("dial failed to %s: %w", destAddr, err)
	}
	defer targetConn.Close()

	clientBuffered := tunnel.NewBufferedConn(reader, clientConn)

	// Direct pass-through for banking, government, or captive portals
	if action == rules.ActionDirect || group == probe.GroupSafe {
		tunnel.Pipe(clientBuffered, targetConn)
		return nil
	}

	// Read initial payload (e.g. TLS ClientHello) with safety bounds
	initialPayload, err := tunnel.ReadInitialPayload(reader)
	if err == nil && len(initialPayload) > 0 {
		info := dpi.ParsePacket(initialPayload)
		_ = o.Fallback.ApplyWithFallback(group, targetConn, initialPayload, info)
	}

	// Stream bidirectional traffic
	tunnel.Pipe(clientBuffered, targetConn)
	return nil
}

// HandleHTTP handles plain HTTP proxy requests
func (o *Orchestrator) HandleHTTP(clientConn net.Conn, reader *bufio.Reader, req *http.Request, targetHost, targetPort string) error {
	destAddr, action, err := o.resolveDestination(targetHost, targetPort)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", targetHost, err)
	}

	group := probe.ClassifyDomain(targetHost)

	targetConn, err := net.DialTimeout("tcp", destAddr, o.DialTimeout)
	if err != nil {
		return fmt.Errorf("dial failed to %s: %w", destAddr, err)
	}
	defer targetConn.Close()

	clientBuffered := tunnel.NewBufferedConn(reader, clientConn)

	isWS := strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
	if isWS || action == rules.ActionDirect || group == probe.GroupSafe {
		_ = req.Write(targetConn)
		tunnel.Pipe(clientBuffered, targetConn)
		return nil
	}

	// Serialize plain HTTP request
	var b strings.Builder
	_ = req.Write(&b)
	reqBuf := []byte(b.String())

	info := dpi.ParsePacket(reqBuf)
	_ = o.Fallback.ApplyWithFallback(group, targetConn, reqBuf, info)

	tunnel.Pipe(clientBuffered, targetConn)
	return nil
}

// resolveDestination evaluates the routing action and resolves the target IP
func (o *Orchestrator) resolveDestination(targetHost, targetPort string) (string, rules.Action, error) {
	action := o.Rules.Evaluate(targetHost)

	var resolvedIP string
	if action == rules.ActionDirect {
		// Captive portal / banking: resolve directly with system DNS
		ips, err := net.DefaultResolver.LookupHost(context.Background(), targetHost)
		if err == nil && len(ips) > 0 {
			resolvedIP = ips[0]
		} else {
			resolvedIP = targetHost
		}
	} else {
		// Censored or general: use DoH multi-tier fallback
		ip, err := o.Resolver.Resolve(context.Background(), targetHost)
		if err == nil && ip != "" && !dns.IsPoisonedIP(ip) {
			resolvedIP = ip
		} else {
			resolvedIP = targetHost
		}
	}

	return net.JoinHostPort(resolvedIP, targetPort), action, nil
}

func (o *Orchestrator) ensureKernelRunning(targetHost string) {
	if rules.IsAntiCheatProcess(targetHost) {
		return
	}
	o.mu.Lock()
	if !o.KernelActive {
		o.KernelActive = true
		o.mu.Unlock()
		_ = divert.Start()
		return
	}
	o.mu.Unlock()
}
