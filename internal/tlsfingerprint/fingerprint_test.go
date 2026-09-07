package tlsfingerprint

import (
	"strings"
	"testing"
)

func TestJA4Calculation(t *testing.T) {
	ciphers := []uint16{0x1301, 0x1302, 0x1303}
	exts := []uint16{0x0000, 0x0010, 0x002b}

	ja4 := CalculateJA4(true, 0x0304, true, ciphers, exts, "h2")

	// Must start with t13d0303h2_
	expectedPrefix := "t13d0303h2_"
	if !strings.HasPrefix(ja4, expectedPrefix) {
		t.Errorf("expected prefix %s, got %s", expectedPrefix, ja4)
	}

	parts := strings.Split(ja4, "_")
	if len(parts) != 3 {
		t.Fatalf("JA4 must consist of 3 parts separated by underscores, got %d", len(parts))
	}
	if len(parts[1]) != 12 || len(parts[2]) != 12 {
		t.Errorf("hashes must be 12 hex chars each, got %s and %s", parts[1], parts[2])
	}
}

func TestBrowserImpersonator(t *testing.T) {
	chrome := NewImpersonator(ProfileChrome130)
	cSig := chrome.GenerateSignature(true, true)
	if !strings.HasPrefix(cSig, "t13d1514h2_") {
		t.Errorf("expected Chrome signature prefix t13d1514h2_, got %s", cSig)
	}

	safari := NewImpersonator(ProfileSafari18)
	sSig := safari.GenerateSignature(true, true)
	if !strings.HasPrefix(sSig, "t13d1314h2_") {
		t.Errorf("expected Safari signature prefix t13d1314h2_, got %s", sSig)
	}
}
