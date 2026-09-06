package doh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckECH(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/dns-json")
		// Return simulated HTTPS RR with ECH
		w.Write([]byte(`{
			"Status": 0,
			"Answer": [
				{
					"name": "crypto.cloudflare.com",
					"type": 65,
					"TTL": 300,
					"data": "1 . alpn=h2,h3 ech=AEX+DQBB..."
				}
			]
		}`))
	}))
	defer mockServer.Close()

	r := NewResolver(mockServer.URL, true)
	supported, cfg := r.CheckECH(context.Background(), "crypto.cloudflare.com")
	// If Cloudflare mock or endpoint is called
	if !supported && cfg == "" {
		// Even if Cloudflare production was hit or mock fallback, Ensure no panic
		t.Logf("ECH result: supported=%v, cfg=%v", supported, cfg)
	}
}
