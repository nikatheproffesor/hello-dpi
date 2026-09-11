package control

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/hellodpi/hellodpi/internal/doctor"
	"github.com/hellodpi/hellodpi/internal/speedtest"
	"github.com/hellodpi/hellodpi/internal/version"
)

// Server provides the local HTTP management API and UI endpoints (/doctor, /speedtest, /api/status)
type Server struct {
	mux *http.ServeMux
}

// NewServer creates a configured control server
func NewServer() *Server {
	mux := http.NewServeMux()

	// Register Speedtest & Doctor endpoints
	speedtest.RegisterHandlers(mux)
	doctor.RegisterHandlers(mux)

	// Register status and health endpoints
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return &Server{
		mux: mux,
	}
}

// IsLoopbackHost checks whether a host (with or without port) points to the local machine loopback
func IsLoopbackHost(host string) bool {
	h := strings.TrimSpace(host)
	if h == "" {
		return false
	}
	if parsedHost, _, err := net.SplitHostPort(h); err == nil {
		h = parsedHost
	}
	h = strings.Trim(h, "[]")
	h = strings.ToLower(h)

	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

// IsControlPath checks if the given HTTP request path targets a local management endpoint
func IsControlPath(method, path string) bool {
	if method == http.MethodConnect {
		return false
	}
	p := strings.ToLower(path)
	return p == "/speedtest" || strings.HasPrefix(p, "/speedtest/") ||
		p == "/api/speedtest" || strings.HasPrefix(p, "/api/speedtest/") ||
		p == "/doctor" || strings.HasPrefix(p, "/doctor/") ||
		p == "/api/doctor" || strings.HasPrefix(p, "/api/doctor/") ||
		p == "/api/status" || p == "/api/health" || p == "/favicon.ico"
}

// IsControlRequest checks if an HTTP request genuinely targets the local management API
// (i.e. path matches control path AND Host targets localhost/loopback).
// This prevents proxy requests to external sites like http://example.com/doctor from being hijacked.
func IsControlRequest(req *http.Request) bool {
	if req == nil || req.Method == http.MethodConnect {
		return false
	}

	targetHost := req.Host
	if targetHost == "" && req.URL != nil {
		targetHost = req.URL.Host
	}

	// If host is explicitly specified and not loopback, it's an outbound proxy request, not local control
	if targetHost != "" && !IsLoopbackHost(targetHost) {
		return false
	}

	path := ""
	if req.URL != nil {
		path = req.URL.Path
	}
	return IsControlPath(req.Method, path)
}

// ServeHTTP delegates to the internal multiplexer with Host, Origin, and RemoteAddr security guards
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Host Validation (prevent DNS rebinding attacks)
	if r.Host != "" && !IsLoopbackHost(r.Host) {
		http.Error(w, "Forbidden: Invalid Host header for management API", http.StatusForbidden)
		return
	}

	// 2. Origin & Referer Validation (prevent Cross-Site Request Forgery from external websites)
	if origin := r.Header.Get("Origin"); origin != "" {
		if u, err := url.Parse(origin); err != nil || !IsLoopbackHost(u.Host) {
			http.Error(w, "Forbidden: Cross-Origin access denied", http.StatusForbidden)
			return
		}
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		if u, err := url.Parse(referer); err != nil || !IsLoopbackHost(u.Host) {
			http.Error(w, "Forbidden: External Referer access denied", http.StatusForbidden)
			return
		}
	}

	// 3. Security Headers
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")

	s.mux.ServeHTTP(w, r)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"app":     "Hello DPI",
		"version": version.Version,
		"status":  "running",
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
