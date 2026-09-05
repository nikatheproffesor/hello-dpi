package updater

import (
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	cases := []struct {
		remote   string
		current  string
		expected bool
	}{
		{"v2.1.0", "2.0.0", true},
		{"v2.0.1", "2.0.0", true},
		{"v3.0.0", "2.1.0", true},
		{"2.1.0", "2.1.0", false},
		{"v2.0.0", "2.1.0", false},
		{"v1.9.9", "2.0.0", false},
		{"v2.1.0-beta", "2.1.0", false},
		{"v2.1.1", "2.1.0", true},
	}

	for _, tc := range cases {
		actual := IsNewerVersion(tc.remote, tc.current)
		if actual != tc.expected {
			t.Errorf("IsNewerVersion(%q, %q) = %v, expected %v", tc.remote, tc.current, actual, tc.expected)
		}
	}
}

func TestFindMatchingAsset(t *testing.T) {
	assets := []ReleaseAsset{
		{Name: "HelloDPI-Windows.exe", BrowserDownloadURL: "https://example.com/win.exe", Size: 1000},
		{Name: "HelloDPI-macOS.dmg", BrowserDownloadURL: "https://example.com/mac.dmg", Size: 2000},
		{Name: "hellodpi-linux-amd64", BrowserDownloadURL: "https://example.com/linux", Size: 3000},
	}

	matched := findMatchingAsset(assets)
	if matched == nil {
		t.Fatalf("expected matched asset, got nil")
	}
}

func TestFormatBytes(t *testing.T) {
	if FormatBytes(500) != "500 B" {
		t.Errorf("expected '500 B', got %s", FormatBytes(500))
	}
	if FormatBytes(1048576) != "1.0 MB" {
		t.Errorf("expected '1.0 MB', got %s", FormatBytes(1048576))
	}
}

func TestLiveGitHubAPI(t *testing.T) {
	rel, _, err := CheckUpdate()
	if err != nil {
		t.Fatalf("CheckUpdate failed: %v", err)
	}
	if rel.TagName != "v2.1.1" {
		t.Errorf("expected latest tag 'v2.1.1', got %s", rel.TagName)
	}
	if rel.TargetAsset == nil {
		t.Errorf("expected matched TargetAsset, got nil")
	}
}
