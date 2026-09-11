package doctor

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunRepair(t *testing.T) {
	defer ResetNetworkToCleanState()
	report := RunRepair()
	if report == nil {
		t.Fatalf("expected report, got nil")
	}

	if len(report.Steps) < 3 {
		t.Errorf("expected at least 3 diagnostic steps, got %d", len(report.Steps))
	}

	if report.Timestamp == "" {
		t.Errorf("expected timestamp, got empty")
	}

	if len(report.Logs) == 0 {
		t.Errorf("expected diagnostic logs, got 0")
	}
}

func TestHandlers(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHandlers(mux)

	// 1. Dashboard
	reqDashboard := httptest.NewRequest("GET", "/doctor", nil)
	recDashboard := httptest.NewRecorder()
	mux.ServeHTTP(recDashboard, reqDashboard)
	if recDashboard.Code != http.StatusOK {
		t.Errorf("expected status 200 for /doctor, got %d", recDashboard.Code)
	}

	// 2. Trailing slash /doctor/
	reqSlash := httptest.NewRequest("GET", "/doctor/", nil)
	recSlash := httptest.NewRecorder()
	mux.ServeHTTP(recSlash, reqSlash)
	if recSlash.Code != http.StatusOK {
		t.Errorf("expected status 200 for /doctor/, got %d", recSlash.Code)
	}

	// 3. API fix-dns with GET should be rejected with 405 Method Not Allowed
	reqDNSGet := httptest.NewRequest("GET", "/api/doctor/fix-dns", nil)
	recDNSGet := httptest.NewRecorder()
	mux.ServeHTTP(recDNSGet, reqDNSGet)
	if recDNSGet.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET /api/doctor/fix-dns, got %d", recDNSGet.Code)
	}

	// 4. API fix-dns with POST should succeed
	reqDNSPost := httptest.NewRequest("POST", "/api/doctor/fix-dns", nil)
	recDNSPost := httptest.NewRecorder()
	mux.ServeHTTP(recDNSPost, reqDNSPost)
	if recDNSPost.Code != http.StatusOK {
		t.Errorf("expected status 200 for POST /api/doctor/fix-dns, got %d", recDNSPost.Code)
	}

	// 5. Read-only endpoint kernel-status accepts GET
	reqKernelStatus := httptest.NewRequest("GET", "/api/doctor/kernel-status", nil)
	recKernelStatus := httptest.NewRecorder()
	mux.ServeHTTP(recKernelStatus, reqKernelStatus)
	if recKernelStatus.Code != http.StatusOK {
		t.Errorf("expected status 200 for GET /api/doctor/kernel-status, got %d", recKernelStatus.Code)
	}
}
