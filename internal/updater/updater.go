package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hellodpi/hellodpi/internal/version"
)

const (
	githubRepo     = "emreaytekxn/hello-dpi"
	apiURL         = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	requestTimeout = 10 * time.Second
)

// ReleaseAsset represents an asset in a GitHub release
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ReleaseInfo contains information about a GitHub release
type ReleaseInfo struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	HTMLURL     string         `json:"html_url"`
	Assets      []ReleaseAsset `json:"assets"`
	TargetAsset *ReleaseAsset  `json:"-"`
}

// CheckUpdate checks GitHub for newer releases than current version
func CheckUpdate() (*ReleaseInfo, bool, error) {
	// 1. Try standard GitHub API
	rel, isNew, err := checkViaAPI()
	if err == nil {
		return rel, isNew, nil
	}

	// 2. Fallback to GitHub Web 302 Redirect (Immune to GitHub API 60 req/hr rate limits!)
	return checkViaWebRedirect()
}

func checkViaAPI() (*ReleaseInfo, bool, error) {
	client := &http.Client{Timeout: requestTimeout}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "HelloDPI/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to query GitHub releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("github api returned status: %s", resp.Status)
	}

	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, false, fmt.Errorf("failed to parse release json: %w", err)
	}

	// Match asset for current OS/Arch
	rel.TargetAsset = findMatchingAsset(rel.Assets)

	if !IsNewerVersion(rel.TagName, version.Version) {
		return &rel, false, nil
	}

	return &rel, true, nil
}

func checkViaWebRedirect() (*ReleaseInfo, bool, error) {
	client := &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	webURL := "https://github.com/" + githubRepo + "/releases/latest"
	req, err := http.NewRequest("HEAD", webURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", "HelloDPI/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check latest release: %w", err)
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return nil, false, fmt.Errorf("no release redirect location found")
	}

	idx := strings.LastIndex(loc, "/")
	if idx == -1 {
		return nil, false, fmt.Errorf("invalid release location: %s", loc)
	}
	tagName := loc[idx+1:]

	rel := &ReleaseInfo{
		TagName: tagName,
		Name:    "Hello DPI " + tagName,
		HTMLURL: loc,
	}

	var assetName string
	switch runtime.GOOS {
	case "windows":
		assetName = "HelloDPI-Windows.exe"
	case "darwin":
		assetName = "HelloDPI-macOS.dmg"
	default:
		assetName = "hellodpi-linux-amd64"
	}

	rel.TargetAsset = &ReleaseAsset{
		Name:               assetName,
		BrowserDownloadURL: fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", githubRepo, tagName, assetName),
	}

	if !IsNewerVersion(rel.TagName, version.Version) {
		return rel, false, nil
	}

	return rel, true, nil
}

// IsNewerVersion compares remote version tag with current version
func IsNewerVersion(remoteTag, currentVer string) bool {
	cleanRemote := strings.TrimPrefix(strings.TrimSpace(remoteTag), "v")
	cleanCurrent := strings.TrimPrefix(strings.TrimSpace(currentVer), "v")

	rParts := parseVersionParts(cleanRemote)
	cParts := parseVersionParts(cleanCurrent)

	for i := 0; i < len(rParts) && i < len(cParts); i++ {
		if rParts[i] > cParts[i] {
			return true
		}
		if rParts[i] < cParts[i] {
			return false
		}
	}
	return len(rParts) > len(cParts)
}

func parseVersionParts(v string) []int {
	parts := strings.Split(v, ".")
	var res []int
	for _, p := range parts {
		num := 0
		for _, r := range p {
			if r >= '0' && r <= '9' {
				num = num*10 + int(r-'0')
			} else {
				break
			}
		}
		res = append(res, num)
	}
	return res
}

func findMatchingAsset(assets []ReleaseAsset) *ReleaseAsset {
	osName := runtime.GOOS

	for _, a := range assets {
		name := strings.ToLower(a.Name)
		switch osName {
		case "windows":
			if strings.HasSuffix(name, ".exe") {
				assetCopy := a
				return &assetCopy
			}
		case "darwin":
			if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".dmg") || strings.Contains(name, "darwin") || strings.Contains(name, "macos") {
				assetCopy := a
				return &assetCopy
			}
		case "linux":
			if strings.Contains(name, "linux") && !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") {
				assetCopy := a
				return &assetCopy
			}
		}
	}

	// Fallback to first asset if any
	if len(assets) > 0 {
		assetCopy := assets[0]
		return &assetCopy
	}
	return nil
}

// ApplyUpdate downloads the new binary and performs seamless in-place replacement
func ApplyUpdate(rel *ReleaseInfo, onProgress func(percent int)) error {
	if rel.TargetAsset == nil {
		return fmt.Errorf("no compatible download asset found for %s", runtime.GOOS)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine current executable path: %w", err)
	}
	currentExe, _ = filepath.EvalSymlinks(currentExe)

	// Create temp file for download in the same directory to allow atomic renames
	dir := filepath.Dir(currentExe)
	tempFile, err := os.CreateTemp(dir, "hellodpi-update-*")
	if err != nil {
		// Fallback to system temp directory
		tempFile, err = os.CreateTemp("", "hellodpi-update-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath) // Clean up temp file on failure

	// Download new binary with progress
	resp, err := http.Get(rel.TargetAsset.BrowserDownloadURL)
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("download error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tempFile.Close()
		return fmt.Errorf("download returned status: %s", resp.Status)
	}

	totalSize := resp.ContentLength
	if totalSize <= 0 {
		totalSize = rel.TargetAsset.Size
	}

	var downloaded int64
	buf := make([]byte, 32*1024)
	lastPercent := -1

	for {
		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := tempFile.Write(buf[:n]); wErr != nil {
				tempFile.Close()
				return fmt.Errorf("write error: %w", wErr)
			}
			downloaded += int64(n)
			if totalSize > 0 && onProgress != nil {
				percent := int((downloaded * 100) / totalSize)
				if percent != lastPercent {
					lastPercent = percent
					onProgress(percent)
				}
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			tempFile.Close()
			return fmt.Errorf("stream read error: %w", rErr)
		}
	}
	tempFile.Close()

	// Ensure downloaded file is executable
	_ = os.Chmod(tempPath, 0755)

	// Perform atomic in-place replacement
	oldExePath := currentExe + ".old"
	_ = os.Remove(oldExePath) // remove previous backup if exists

	// Step 1: Rename currently running binary to .old (Windows & Unix allow renaming active binaries!)
	if err := os.Rename(currentExe, oldExePath); err != nil {
		return fmt.Errorf("failed renaming current binary to .old: %w", err)
	}

	// Step 2: Move new binary into original location
	if err := os.Rename(tempPath, currentExe); err != nil {
		// Rollback if failed
		_ = os.Rename(oldExePath, currentExe)
		return fmt.Errorf("failed installing new binary: %w", err)
	}

	// Step 3: Remove .old binary (Unix removes immediately; Windows removes on next boot or exit)
	_ = os.Remove(oldExePath)

	return nil
}

// RestartApp launches the newly updated executable and terminates the old process
func RestartApp() error {
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, _ = filepath.EvalSymlinks(currentExe)

	// Check if running inside macOS .app bundle
	if runtime.GOOS == "darwin" && strings.Contains(currentExe, ".app/Contents/MacOS/") {
		appPath := currentExe[:strings.Index(currentExe, ".app")+4]
		cmd := exec.Command("open", "-n", appPath)
		if err := cmd.Start(); err != nil {
			return err
		}
		os.Exit(0)
	}

	cmd := exec.Command(currentExe, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil
}

// FormatBytes formats byte sizes into human readable strings
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return strconv.FormatInt(b, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
