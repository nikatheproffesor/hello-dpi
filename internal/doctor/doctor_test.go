package doctor

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunRepair(t *testing.T) {
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

	// 3. API fix-dns
	reqDNS := httptest.NewRequest("GET", "/api/doctor/fix-dns", nil)
	recDNS := httptest.NewRecorder()
	mux.ServeHTTP(recDNS, reqDNS)
	if recDNS.Code != http.StatusOK {
		t.Errorf("expected status 200 for /api/doctor/fix-dns, got %d", recDNS.Code)
	}
}
