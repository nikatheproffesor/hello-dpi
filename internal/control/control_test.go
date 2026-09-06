package control

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsControlPath(t *testing.T) {
	if !IsControlPath("GET", "/doctor") {
		t.Errorf("Expected /doctor to be control path")
	}
	if !IsControlPath("GET", "/speedtest") {
		t.Errorf("Expected /speedtest to be control path")
	}
	if !IsControlPath("GET", "/api/status") {
		t.Errorf("Expected /api/status to be control path")
	}
	if IsControlPath("CONNECT", "/doctor") {
		t.Errorf("CONNECT should never be treated as control path")
	}
	if IsControlPath("GET", "https://discord.com/") {
		t.Errorf("External site should not be control path")
	}
}

func TestControlServerStatus(t *testing.T) {
	srv := NewServer()

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()

	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}
