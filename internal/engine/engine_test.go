package engine

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/rules"
)

func TestOrchestratorDirectRouting(t *testing.T) {
	orch := NewOrchestrator(Config{
		StrategyName: string(dpi.SplitTLS),
		DelayMs:      1,
	})

	// Localhost / private IP must route as ActionDirect
	_, action, err := orch.resolveDestination("127.0.0.1", "80")
	if err != nil {
		t.Fatalf("resolveDestination failed: %v", err)
	}
	if action != rules.ActionDirect {
		t.Errorf("Expected 127.0.0.1 to be ActionDirect, got %d", action)
	}
}

func TestOrchestratorUpdateStrategy(t *testing.T) {
	orch := NewOrchestrator(Config{
		StrategyName: string(dpi.SplitTLS),
		DelayMs:      5,
	})

	orch.UpdateStrategy(string(dpi.SplitSNI), 3, 2)

	orch.mu.RLock()
	stratName := orch.Strategy.Name()
	splitOffset := orch.SplitOffset
	delay := orch.DelayMs
	orch.mu.RUnlock()

	if stratName != string(dpi.SplitSNI) {
		t.Errorf("Expected strategy '%s', got '%s'", dpi.SplitSNI, stratName)
	}
	if splitOffset != 3 || delay != 2 {
		t.Errorf("Expected offset 3 and delay 2ms, got offset=%d, delay=%d", splitOffset, delay)
	}
}

func TestOrchestratorHandleTunnelDirect(t *testing.T) {
	orch := NewOrchestrator(Config{
		StrategyName: string(dpi.SplitTLS),
		DialTimeout:  2 * time.Second,
	})

	// Setup mock echo server for direct pass
	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer echoListener.Close()

	_, port, _ := net.SplitHostPort(echoListener.Addr().String())

	go func() {
		conn, err := echoListener.Accept()
		if err == nil {
			defer conn.Close()
			buf := make([]byte, 4)
			_, _ = io.ReadFull(conn, buf)
			_, _ = conn.Write([]byte("ECHO"))
		}
	}()

	clientA, clientB := net.Pipe()
	defer clientA.Close()
	defer clientB.Close()

	reader := bufio.NewReader(bytes.NewReader([]byte("PING")))

	done := make(chan error)
	go func() {
		done <- orch.HandleTunnel(clientB, reader, "127.0.0.1", port, nil)
	}()

	reply := make([]byte, 4)
	_ = clientA.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadFull(clientA, reply)
	if err != nil {
		t.Fatalf("Failed reading echo reply: %v", err)
	}
	if string(reply) != "ECHO" {
		t.Fatalf("Expected ECHO, got %s", string(reply))
	}
}

func TestOrchestratorHandleHTTPDirect(t *testing.T) {
	orch := NewOrchestrator(Config{
		StrategyName: string(dpi.SplitAuto),
		DialTimeout:  2 * time.Second,
	})

	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer echoListener.Close()

	_, port, _ := net.SplitHostPort(echoListener.Addr().String())

	go func() {
		conn, err := echoListener.Accept()
		if err == nil {
			defer conn.Close()
			req, _ := http.ReadRequest(bufio.NewReader(conn))
			if req != nil {
				_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
			}
		}
	}()

	clientA, clientB := net.Pipe()
	defer clientA.Close()
	defer clientB.Close()

	req, _ := http.NewRequest("GET", "http://127.0.0.1:"+port+"/", nil)
	reader := bufio.NewReader(bytes.NewReader([]byte{}))

	go func() {
		_ = orch.HandleHTTP(clientB, reader, req, "127.0.0.1", port)
	}()

	clientReader := bufio.NewReader(clientA)
	resp, err := http.ReadResponse(clientReader, req)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}
}

type failStrategy struct{}

func (f *failStrategy) Name() string { return "failing" }
func (f *failStrategy) Apply(conn net.Conn, data []byte, info dpi.ParsedInfo) error {
	return fmt.Errorf("simulated network failure")
}

func TestFallbackTracker(t *testing.T) {
	fallbackCh := make(chan struct{}, 1)
	ft := NewFallbackTracker(func(group probe.DomainGroup, fromStrat, toStrat string) {
		select {
		case fallbackCh <- struct{}{}:
		default:
		}
	})

	ft.SetGroupStrategies(probe.GroupDiscord, []dpi.BypassStrategy{
		&failStrategy{},
		dpi.DefaultStrategy(),
	})

	cA, cB := net.Pipe()
	defer cA.Close()
	defer cB.Close()

	go func() {
		buf := make([]byte, 1024)
		for {
			_, err := cA.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	payload := []byte("TEST_PAYLOAD")
	info := dpi.ParsedInfo{}

	// First failure
	_ = ft.ApplyWithFallback(probe.GroupDiscord, cB, payload, info)
	if streak := ft.GetFailureStreak(probe.GroupDiscord); streak != 1 {
		t.Errorf("Expected failure streak 1, got %d", streak)
	}

	// Second failure triggers switch
	_ = ft.ApplyWithFallback(probe.GroupDiscord, cB, payload, info)
	select {
	case <-fallbackCh:
		// Succeeded
	case <-time.After(1 * time.Second):
		t.Errorf("Timed out waiting for fallback callback")
	}

	// Active strategy should now be DefaultStrategy
	active := ft.GetActiveStrategy(probe.GroupDiscord)
	if active.Name() != dpi.DefaultStrategy().Name() {
		t.Errorf("Expected switched active strategy, got %s", active.Name())
	}
}
