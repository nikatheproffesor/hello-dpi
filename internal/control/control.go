package control

import (
	"encoding/json"
	"net/http"
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

// ServeHTTP delegates to the internal multiplexer
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
