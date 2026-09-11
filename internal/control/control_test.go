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

func TestIsControlRequest(t *testing.T) {
	// 1. Local management request with loopback host -> true
	reqLocal := httptest.NewRequest("GET", "http://127.0.0.1:8080/doctor", nil)
	if !IsControlRequest(reqLocal) {
		t.Errorf("Expected local request to 127.0.0.1:8080/doctor to be recognized as control request")
	}

	// 2. Outbound proxy request to external domain /doctor -> false (must not hijack external traffic)
	reqExternal := httptest.NewRequest("GET", "http://example.com/doctor", nil)
	if IsControlRequest(reqExternal) {
		t.Errorf("External proxy request to example.com/doctor MUST NOT be treated as local control request")
	}

	// 3. Outbound HTTPS CONNECT -> false
	reqConnect := httptest.NewRequest("CONNECT", "example.com:443", nil)
	if IsControlRequest(reqConnect) {
		t.Errorf("CONNECT request must not be treated as control request")
	}
}

func TestControlServerSecurity(t *testing.T) {
	srv := NewServer()

	// 1. Valid local request with loopback Host
	reqLocal := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/status", nil)
	reqLocal.Host = "127.0.0.1:8080"
	wLocal := httptest.NewRecorder()
	srv.ServeHTTP(wLocal, reqLocal)
	if wLocal.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for valid local request, got %d", wLocal.Code)
	}

	// 2. Request with foreign Host header (DNS rebinding attempt) -> 403 Forbidden
	reqRebind := httptest.NewRequest("GET", "/api/status", nil)
	reqRebind.Host = "attacker.com"
	wRebind := httptest.NewRecorder()
	srv.ServeHTTP(wRebind, reqRebind)
	if wRebind.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 for foreign Host header, got %d", wRebind.Code)
	}

	// 3. Request with external Origin (CSRF attempt from external webpage) -> 403 Forbidden
	reqCSRF := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/status", nil)
	reqCSRF.Host = "127.0.0.1:8080"
	reqCSRF.Header.Set("Origin", "https://malicious-website.com")
	wCSRF := httptest.NewRecorder()
	srv.ServeHTTP(wCSRF, reqCSRF)
	if wCSRF.Code != http.StatusForbidden {
		t.Fatalf("Expected status 403 for malicious Origin, got %d", wCSRF.Code)
	}

	// 4. Request with valid local Origin -> 200 OK
	reqValidOrigin := httptest.NewRequest("GET", "http://127.0.0.1:8080/api/status", nil)
	reqValidOrigin.Host = "127.0.0.1:8080"
	reqValidOrigin.Header.Set("Origin", "http://127.0.0.1:8080")
	wValidOrigin := httptest.NewRecorder()
	srv.ServeHTTP(wValidOrigin, reqValidOrigin)
	if wValidOrigin.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for valid local Origin, got %d", wValidOrigin.Code)
	}
}
