package updater

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
		{Name: "HelloDPI-macOS.zip", BrowserDownloadURL: "https://example.com/mac.zip", Size: 1500},
		{Name: "hellodpi-linux-amd64", BrowserDownloadURL: "https://example.com/linux", Size: 3000},
	}

	matched := findMatchingAsset(assets)
	if matched == nil {
		t.Fatalf("expected matched asset, got nil")
	}

	// On darwin, it must specifically prefer .zip over .dmg for bundle extraction!
	if strings.Contains(matched.Name, "macOS") && !strings.HasSuffix(matched.Name, ".zip") {
		t.Errorf("Expected macOS to select .zip asset, got %s", matched.Name)
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
	if !strings.HasPrefix(rel.TagName, "v") {
		t.Errorf("expected tag starting with 'v', got %s", rel.TagName)
	}
	if rel.TargetAsset == nil {
		t.Errorf("expected matched TargetAsset, got nil")
	}
}

func TestApplyMacOSZipUpdate(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping macOS-specific zip update test on non-darwin OS")
	}

	tempDir := t.TempDir()

	// 1. Create a simulated target bundle (v1)
	targetApp := filepath.Join(tempDir, "Hello DPI.app")
	targetExe := filepath.Join(targetApp, "Contents", "MacOS", "Hello DPI")
	_ = os.MkdirAll(filepath.Dir(targetExe), 0755)
	_ = os.WriteFile(targetExe, []byte("v1-binary"), 0755)

	// 2. Create a source bundle (v2) to zip
	srcDir := filepath.Join(tempDir, "source")
	srcApp := filepath.Join(srcDir, "Hello DPI.app")
	srcExe := filepath.Join(srcApp, "Contents", "MacOS", "Hello DPI")
	_ = os.MkdirAll(filepath.Dir(srcExe), 0755)
	_ = os.WriteFile(srcExe, []byte("v2-binary"), 0755)

	// 3. Zip source bundle
	zipPath := filepath.Join(tempDir, "update.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed creating zip: %v", err)
	}
	zw := zip.NewWriter(zipFile)
	w, err := zw.Create("Hello DPI.app/Contents/MacOS/Hello DPI")
	if err != nil {
		t.Fatalf("failed creating zip entry: %v", err)
	}
	_, _ = w.Write([]byte("v2-binary"))
	_ = zw.Close()
	_ = zipFile.Close()

	// 4. Apply update
	if err := applyMacOSZipUpdate(zipPath, targetExe); err != nil {
		t.Fatalf("applyMacOSZipUpdate failed: %v", err)
	}

	// 5. Verify updated content
	content, err := os.ReadFile(targetExe)
	if err != nil {
		t.Fatalf("failed reading target executable: %v", err)
	}
	if string(content) != "v2-binary" {
		t.Errorf("expected 'v2-binary', got '%s'", string(content))
	}
}

func TestApplyMacOSDmgUpdate(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping macOS-specific DMG update test on non-darwin OS")
	}

	dmgPath := "../../bin/HelloDPI-macOS.dmg"
	if _, err := os.Stat(dmgPath); os.IsNotExist(err) {
		t.Skip("bin/HelloDPI-macOS.dmg does not exist, skipping DMG test")
	}

	tempDir := t.TempDir()
	targetApp := filepath.Join(tempDir, "Hello DPI.app")
	targetExe := filepath.Join(targetApp, "Contents", "MacOS", "Hello DPI")
	_ = os.MkdirAll(filepath.Dir(targetExe), 0755)
	_ = os.WriteFile(targetExe, []byte("old-binary"), 0755)

	if err := applyMacOSDmgUpdate(dmgPath, targetExe); err != nil {
		t.Fatalf("applyMacOSDmgUpdate failed: %v", err)
	}

	// Verify target executable is now a real Mach-O binary from the DMG
	stat, err := os.Stat(targetExe)
	if err != nil {
		t.Fatalf("failed stating target executable: %v", err)
	}
	if stat.Size() < 1000000 {
		t.Errorf("expected real Mach-O binary (>1MB), got size %d", stat.Size())
	}
}
