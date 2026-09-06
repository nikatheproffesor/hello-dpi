package engine

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/hellodpi/hellodpi/internal/dpi"
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
		done <- orch.HandleTunnel(clientB, reader, "127.0.0.1", port)
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
