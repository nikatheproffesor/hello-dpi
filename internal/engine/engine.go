package engine

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/divert"
	"github.com/hellodpi/hellodpi/internal/dns"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/ech"
	"github.com/hellodpi/hellodpi/internal/heuristic"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/rules"
	"github.com/hellodpi/hellodpi/internal/tlsfingerprint"
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
	Heuristic    *heuristic.Engine
	ECH          *ech.Manager
	Fingerprint  *tlsfingerprint.Impersonator
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
	ft := NewFallbackTracker(nil)
	ft.SetGroupChains(map[probe.DomainGroup]string{
		probe.GroupWeb:     cfg.StrategyName,
		probe.GroupDiscord: cfg.StrategyName,
		probe.GroupRoblox:  cfg.StrategyName,
	}, nil)

	return &Orchestrator{
		Rules:       rules.NewEngine(),
		Resolver:    dns.NewResolver(cfg.DoHEndpoint, cfg.EnableDoH),
		Strategy:    strat,
		Fallback:    ft,
		Heuristic:   heuristic.NewEngine(),
		ECH:         ech.NewManager(),
		Fingerprint: tlsfingerprint.NewImpersonator(tlsfingerprint.ProfileChrome130),
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
		probe.GroupRoblox:  name,
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
// The onConnected callback is invoked immediately after targetConn is successfully dialed,
// allowing the proxy server to send the protocol success response (e.g. 200 Connection Established)
// before reading payload, completely preventing deadlock while avoiding premature success responses.
func (o *Orchestrator) HandleTunnel(clientConn net.Conn, reader *bufio.Reader, targetHost, targetPort string, onConnected func() error) error {
	destAddr, action, err := o.resolveDestination(targetHost, targetPort)
	if err != nil {
		return fmt.Errorf("failed to resolve target %s: %w", targetHost, err)
	}

	group := probe.ClassifyDomain(targetHost)

	// Dial target server first
	targetConn, err := net.DialTimeout("tcp", destAddr, o.DialTimeout)
	if err != nil {
		return fmt.Errorf("dial failed to %s: %w", destAddr, err)
	}
	defer targetConn.Close()

	if onConnected != nil {
		if err := onConnected(); err != nil {
			return err
		}
	}

	clientBuffered := tunnel.NewBufferedConn(reader, clientConn)

	// Direct pass-through for banking, government, or captive portals
	if action == rules.ActionDirect || group == probe.GroupSafe {
		tunnel.Pipe(clientBuffered, targetConn)
		return nil
	}

	// For server-first protocols (e.g. SSH 22, SMTP 25, FTP 21), don't block waiting for client payload
	isServerFirst := targetPort == "21" || targetPort == "22" || targetPort == "25" || targetPort == "110" || targetPort == "143"
	if isServerFirst {
		tunnel.Pipe(clientBuffered, targetConn)
		return nil
	}

	// Read initial payload (e.g. TLS ClientHello) with safety bounds
	initialPayload, err := tunnel.ReadInitialPayload(reader)
	if err == nil && len(initialPayload) > 0 {
		start := time.Now()
		info := dpi.ParsePacket(initialPayload)

		// Check if domain requires ECH encapsulation
		_, _, useECH := o.Heuristic.GetOptimalParameters(targetHost)
		if useECH {
			if echCfg, ok := o.ECH.GetECHConfig(targetHost); ok {
				if outerHello, err := ech.EncapsulateOuterClientHello(initialPayload, echCfg, ""); err == nil {
					initialPayload = outerHello
					info = dpi.ParsePacket(initialPayload)
				}
			}
		}

		applyErr := o.Fallback.ApplyWithFallback(group, targetConn, initialPayload, info)
		rtt := time.Since(start).Milliseconds()
		if applyErr != nil {
			o.Heuristic.RecordHandshakeFailure(targetHost, false)
		} else {
			o.Heuristic.RecordHandshakeSuccess(targetHost, rtt)
		}
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
		if err := req.Write(targetConn); err != nil {
			return err
		}
		tunnel.Pipe(clientBuffered, targetConn)
		return nil
	}

	// For plain HTTP requests, serialize headers separately from body to avoid OOM memory exhaustion on large uploads
	reqHeaderOnly := *req
	reqHeaderOnly.Body = nil
	var hdrBuf bytes.Buffer
	if err := reqHeaderOnly.Write(&hdrBuf); err != nil {
		return fmt.Errorf("failed to serialize HTTP headers: %w", err)
	}

	info := dpi.ParsePacket(hdrBuf.Bytes())
	if err := o.Fallback.ApplyWithFallback(group, targetConn, hdrBuf.Bytes(), info); err != nil {
		return fmt.Errorf("failed to apply bypass to HTTP headers: %w", err)
	}

	// Stream HTTP request body if present directly to remote peer
	if req.Body != nil {
		defer req.Body.Close()
		if _, err := io.Copy(targetConn, req.Body); err != nil {
			return fmt.Errorf("failed to stream HTTP body: %w", err)
		}
	}

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

func (o *Orchestrator) EnsureKernelRunning() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.KernelActive {
		if err := divert.Start(); err != nil {
			return err
		}
		o.KernelActive = true
	}
	return nil
}

func (o *Orchestrator) StopKernel() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.KernelActive {
		err := divert.Stop()
		o.KernelActive = false
		return err
	}
	return nil
}
