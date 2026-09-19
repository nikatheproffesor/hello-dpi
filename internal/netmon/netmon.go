package netmon

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/sysproxy"
)


// NetworkState describes the currently detected network environment
type NetworkState struct {
	PrimaryInterface string    `json:"primary_interface"`
	LocalIP          string    `json:"local_ip"`
	CaptivePortal    bool      `json:"captive_portal"`
	Timestamp        time.Time `json:"timestamp"`
}

// Monitor periodically watches network interface changes, sleep/wake, and captive portals
type Monitor struct {
	mu           sync.RWMutex
	lastState    NetworkState
	proxyHost    string
	proxyPort    int
	proxyActive  bool
	onChange     func(oldState, newState NetworkState)
	stopCh       chan struct{}
	pollInterval time.Duration
}

// NewMonitor initializes a network state monitor
func NewMonitor(proxyHost string, proxyPort int, onChange func(oldState, newState NetworkState)) *Monitor {
	m := &Monitor{
		proxyHost:    proxyHost,
		proxyPort:    proxyPort,
		onChange:     onChange,
		pollInterval: 5 * time.Second,
		stopCh:       make(chan struct{}),
	}
	m.lastState = m.inspect()
	return m
}

// SetProxyState updates whether system proxy should be re-applied on network interface changes
func (m *Monitor) SetProxyState(active bool) {
	m.mu.Lock()
	m.proxyActive = active
	m.mu.Unlock()
}

// Start begins periodic background polling
func (m *Monitor) Start() {
	go func() {
		ticker := time.NewTicker(m.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.check()
			}
		}
	}()
}

// Stop terminates background monitoring
func (m *Monitor) Stop() {
	close(m.stopCh)
}

func (m *Monitor) check() {
	currentState := m.inspect()

	m.mu.Lock()
	oldState := m.lastState
	changed := oldState.LocalIP != currentState.LocalIP || oldState.PrimaryInterface != currentState.PrimaryInterface
	captiveChanged := oldState.CaptivePortal != currentState.CaptivePortal
	m.lastState = currentState
	proxyActive := m.proxyActive
	m.mu.Unlock()

	if changed {
		log.Printf("[Hello DPI NetMon] Network change detected: %s (%s) -> %s (%s)",
			oldState.PrimaryInterface, oldState.LocalIP,
			currentState.PrimaryInterface, currentState.LocalIP)

		// Re-apply system proxy to the new active interface only if proxy was enabled and NOT in captive portal
		if proxyActive && m.proxyPort > 0 && !currentState.CaptivePortal {
			log.Printf("[Hello DPI NetMon] Re-applying system proxy to new network interface...")
			_ = sysproxy.SetSystemProxy(m.proxyHost, m.proxyPort)
		}

		if m.onChange != nil {
			go m.onChange(oldState, currentState)
		}
	} else if captiveChanged {
		if currentState.CaptivePortal {
			log.Printf("[Hello DPI NetMon] Captive portal / login page detected (GSB / KYK / Hotel WiFi).")
		} else {
			log.Printf("[Hello DPI NetMon] Captive portal cleared. Normal internet connectivity restored.")
		}
		if m.onChange != nil {
			go m.onChange(oldState, currentState)
		}
	}
}

func (m *Monitor) inspect() NetworkState {
	st := NetworkState{
		Timestamp: time.Now(),
	}

	// Determine primary outbound local IP
	conn, err := net.DialTimeout("udp", "8.8.8.8:80", 500*time.Millisecond)
	if err == nil {
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		st.LocalIP = localAddr.IP.String()
		_ = conn.Close()
	}

	// Match IP to interface
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, _ := iface.Addrs()
			for _, a := range addrs {
				if ipNet, ok := a.(*net.IPNet); ok {
					if ipNet.IP.String() == st.LocalIP {
						st.PrimaryInterface = iface.Name
						break
					}
				}
			}
			if st.PrimaryInterface != "" {
				break
			}
		}
	}

	// Check captive portal (multi-endpoint direct probe)
	st.CaptivePortal = checkCaptivePortal()
	return st
}

// checkCaptivePortal detects whether HTTP requests are being redirected (e.g. KYK / GSB login)
func checkCaptivePortal() bool {
	return checkCaptivePortalEndpoints("http://connectivitycheck.gstatic.com/generate_204", "http://captive.apple.com/hotspot-detect.html")
}

func checkCaptivePortalEndpoints(googleURL, appleURL string) bool {
	// Probe with direct transport (no proxy) to detect local gateway captive redirects
	directTransport := &http.Transport{
		Proxy:                 nil,
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 1500 * time.Millisecond,
	}

	client := &http.Client{
		Timeout:   1500 * time.Millisecond,
		Transport: directTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Google generate_204 check
	if googleURL != "" {
		ctx1, cancel1 := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel1()
		req1, err := http.NewRequestWithContext(ctx1, "GET", googleURL, nil)
		if err == nil {
			resp1, err := client.Do(req1)
			if err == nil {
				defer resp1.Body.Close()
				if resp1.StatusCode == http.StatusNoContent {
					return false // Clean connection, no captive portal
				}
				if resp1.StatusCode == http.StatusFound || resp1.StatusCode == http.StatusMovedPermanently || resp1.StatusCode == http.StatusTemporaryRedirect || resp1.StatusCode == http.StatusOK {
					return true // Redirected to captive portal or returned login page
				}
			}
		}
	}

	// 2. Apple hotspot-detect fallback check
	if appleURL != "" {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		defer cancel2()
		req2, err := http.NewRequestWithContext(ctx2, "GET", appleURL, nil)
		if err == nil {
			resp2, err := client.Do(req2)
			if err == nil {
				defer resp2.Body.Close()
				if resp2.StatusCode == http.StatusOK {
					buf := make([]byte, 128)
					n, _ := resp2.Body.Read(buf)
					if strings.Contains(string(buf[:n]), "Success") {
						return false // Clean connection
					}
					return true // Captive HTML returned instead of Success
				}
				return true // Redirected
			}
		}
	}

	return false
}


