package netmon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMonitor_Inspect(t *testing.T) {
	m := NewMonitor("127.0.0.1", 8080, nil)
	st := m.inspect()
	if st.Timestamp.IsZero() {
		t.Errorf("Expected non-zero timestamp")
	}
}

func TestMonitor_StartStop(t *testing.T) {
	var notified bool
	m := NewMonitor("127.0.0.1", 8080, func(oldState, newState NetworkState) {
		notified = true
	})
	m.pollInterval = 50 * time.Millisecond
	m.Start()
	time.Sleep(120 * time.Millisecond)
	m.Stop()

	// Verify no panic on stop
	_ = notified
}

func TestCaptivePortalDetection_Clean(t *testing.T) {
	// Clean server returns 204 No Content
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	isCaptive := checkCaptivePortalEndpoints(ts.URL, "")
	if isCaptive {
		t.Errorf("Expected clean connection (204 No Content) to NOT be detected as captive portal")
	}
}

func TestCaptivePortalDetection_Redirect(t *testing.T) {
	// GSB WiFi captive portal redirects with HTTP 302
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://wifi.gsb.gov.tr/", http.StatusFound)
	}))
	defer ts.Close()

	isCaptive := checkCaptivePortalEndpoints(ts.URL, "")
	if !isCaptive {
		t.Errorf("Expected redirected connection (302) to BE detected as captive portal")
	}
}

func TestCaptivePortalDetection_LoginPageHTML(t *testing.T) {
	// GSB WiFi captive portal returns HTTP 200 with login HTML instead of 204
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><title>GSB WiFi Portal</title><body>Giris Yapin</body></html>")
	}))
	defer ts.Close()

	isCaptive := checkCaptivePortalEndpoints(ts.URL, "")
	if !isCaptive {
		t.Errorf("Expected 200 OK login page to BE detected as captive portal")
	}
}

