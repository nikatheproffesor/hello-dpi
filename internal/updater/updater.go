package updater

import (
	"archive/zip"
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

	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/version"
)

const (
	githubRepo     = "nikatheproffesor/hello-dpi"
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
		assetName = "HelloDPI-macOS.zip" // Must be ZIP on macOS for automated in-place updates!
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
	arch := runtime.GOARCH

	switch osName {
	case "windows":
		// Look for Windows EXE or ZIP containing Windows binaries
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if (strings.Contains(name, "windows") || strings.Contains(name, "win")) && strings.HasSuffix(name, ".exe") {
				assetCopy := a
				return &assetCopy
			}
		}
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if (strings.Contains(name, "windows") || strings.Contains(name, "win")) && strings.HasSuffix(name, ".zip") {
				assetCopy := a
				return &assetCopy
			}
		}
	case "darwin":
		// Strongly prefer macOS .zip (contains Hello DPI.app bundle)
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if (strings.Contains(name, "macos") || strings.Contains(name, "darwin") || strings.Contains(name, "apple")) && strings.HasSuffix(name, ".zip") {
				assetCopy := a
				return &assetCopy
			}
		}
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if (strings.Contains(name, "macos") || strings.Contains(name, "darwin")) && strings.HasSuffix(name, ".dmg") {
				assetCopy := a
				return &assetCopy
			}
		}
	case "linux":
		// Match linux binaries matching system architecture
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, "linux") && strings.Contains(name, arch) && !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") {
				assetCopy := a
				return &assetCopy
			}
		}
		// Generic linux fallback if arch not in name
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, "linux") && !strings.Contains(name, "arm") && !strings.Contains(name, "mips") && !strings.HasSuffix(name, ".deb") && !strings.HasSuffix(name, ".rpm") {
				assetCopy := a
				return &assetCopy
			}
		}
	}

	// Never return a cross-OS asset as fallback
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

	// Create temp file for download with appropriate extension in os.TempDir()
	ext := filepath.Ext(rel.TargetAsset.Name)
	if ext == "" {
		ext = ".tmp"
	}
	tempFile, err := os.CreateTemp("", "hellodpi-update-*"+ext)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
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

	// macOS update handling:
	// Handle both DMG and ZIP release assets safely. Never treat an archive or disk image as a Mach-O executable.
	if runtime.GOOS == "darwin" {
		lowerName := strings.ToLower(rel.TargetAsset.Name)
		if strings.HasSuffix(lowerName, ".dmg") {
			return applyMacOSDmgUpdate(tempPath, currentExe)
		}
		if strings.HasSuffix(lowerName, ".zip") {
			return applyMacOSZipUpdate(tempPath, currentExe)
		}
		// If running from inside a .app bundle, disallow raw rename of non-archive files
		if strings.Contains(currentExe, ".app") {
			return fmt.Errorf("unsupported update asset format for macOS .app bundle: %s", rel.TargetAsset.Name)
		}
	}

	// Ensure downloaded file is executable
	_ = os.Chmod(tempPath, 0755)

	// Perform atomic in-place replacement (for standalone Windows/Linux binaries or raw CLI)
	oldExePath := currentExe + ".old"
	_ = os.Remove(oldExePath) // remove previous backup if exists

	// Step 1: Rename currently running binary to .old (Windows & Unix allow renaming active binaries!)
	if err := os.Rename(currentExe, oldExePath); err != nil {
		if runtime.GOOS == "darwin" {
			_ = os.Remove(currentExe)
		} else {
			return fmt.Errorf("failed renaming current binary to .old: %w", err)
		}
	}

	// Step 2: Move new binary into original location
	if err := os.Rename(tempPath, currentExe); err != nil {
		if copyErr := copyFile(tempPath, currentExe); copyErr != nil {
			if _, statErr := os.Stat(oldExePath); statErr == nil {
				_ = os.Rename(oldExePath, currentExe)
			}
			return fmt.Errorf("failed installing new binary: %w", copyErr)
		}
	}

	_ = os.Chmod(currentExe, 0755)
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-cr", currentExe).Run()
	}

	// Step 3: Remove .old binary
	_ = os.Remove(oldExePath)

	return nil
}

// applyMacOSDmgUpdate mounts the downloaded disk image, locates the .app bundle, and replaces the target bundle
func applyMacOSDmgUpdate(dmgPath, currentExe string) error {
	mountDir, err := os.MkdirTemp("", "hellodpi-dmg-*")
	if err != nil {
		return fmt.Errorf("failed creating temp mount point: %w", err)
	}
	defer os.RemoveAll(mountDir)

	// Silently attach DMG in read-only and no-browse mode
	cmd := exec.Command("hdiutil", "attach", dmgPath, "-nobrowse", "-readonly", "-mountpoint", mountDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed attaching update disk image: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	defer func() {
		_ = exec.Command("hdiutil", "detach", mountDir, "-force", "-quiet").Run()
	}()

	// Locate .app bundle inside the mounted DMG
	sourceApp := filepath.Join(mountDir, "Hello DPI.app")
	if _, err := os.Stat(sourceApp); err != nil {
		entries, _ := os.ReadDir(mountDir)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".app") {
				sourceApp = filepath.Join(mountDir, e.Name())
				break
			}
		}
	}

	if _, err := os.Stat(sourceApp); err != nil {
		return fmt.Errorf("could not find .app bundle in update disk image")
	}

	return replaceMacOSAppBundle(sourceApp, currentExe)
}

// applyMacOSZipUpdate safely extracts the macOS update zip and replaces the target bundle
func applyMacOSZipUpdate(zipPath, currentExe string) error {
	destTempDir, err := os.MkdirTemp("", "hellodpi-unzip-*")
	if err != nil {
		return fmt.Errorf("failed creating temp unzip directory: %w", err)
	}
	defer os.RemoveAll(destTempDir)

	if err := extractZipArchive(zipPath, destTempDir); err != nil {
		return fmt.Errorf("failed extracting update zip: %w", err)
	}

	// Locate extracted Hello DPI.app bundle
	sourceApp := filepath.Join(destTempDir, "Hello DPI.app")
	if _, err := os.Stat(sourceApp); err != nil {
		entries, _ := os.ReadDir(destTempDir)
		for _, e := range entries {
			if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
				sourceApp = filepath.Join(destTempDir, e.Name())
				break
			}
		}
	}

	if _, err := os.Stat(sourceApp); err != nil {
		return fmt.Errorf("could not find .app bundle in update archive")
	}

	return replaceMacOSAppBundle(sourceApp, currentExe)
}

// replaceMacOSAppBundle atomically replaces the running .app bundle (or CLI binary) with the updated version
func replaceMacOSAppBundle(sourceApp, currentExe string) error {
	// Case 1: Running from within a macOS .app bundle (e.g. /Applications/Hello DPI.app/Contents/MacOS/Hello DPI)
	if appIdx := strings.Index(currentExe, ".app"); appIdx != -1 {
		targetBundle := currentExe[:appIdx+4]

		// To replace an actively running application bundle on macOS APFS:
		// 1. Move the current bundle aside to a temporary backup name.
		//    macOS APFS allows renaming the parent bundle directory even while its child binary is running!
		backupBundle := fmt.Sprintf("%s.old.%d", targetBundle, os.Getpid())
		_ = os.RemoveAll(backupBundle)

		if err := os.Rename(targetBundle, backupBundle); err == nil {
			// Copy the new bundle into the target path using ditto
			// ditto preserves Apple Developer ID codesign, notarization, Mach-O architectures, and metadata
			cmd := exec.Command("ditto", sourceApp, targetBundle)
			if out, dittoErr := cmd.CombinedOutput(); dittoErr != nil {
				// Rollback if ditto failed
				_ = os.RemoveAll(targetBundle)
				_ = os.Rename(backupBundle, targetBundle)
				return fmt.Errorf("failed copying new application bundle: %w (%s)", dittoErr, strings.TrimSpace(string(out)))
			}

			// Clear Gatekeeper quarantine attributes from the newly installed bundle
			_ = exec.Command("xattr", "-cr", targetBundle).Run()

			// Asynchronously remove the backup bundle after this process terminates
			_ = exec.Command("sh", "-c", "sleep 3 && rm -rf \"$1\"", "--", backupBundle).Start()
			return nil
		}

		// Fallback: If bundle directory rename failed, attempt direct ditto overwrite
		cmd := exec.Command("ditto", sourceApp, targetBundle)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed updating application bundle: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		_ = exec.Command("xattr", "-cr", targetBundle).Run()
		return nil
	}

	// Case 2: Standalone CLI binary (e.g. running from /usr/local/bin or ./Hello DPI)
	newInnerExe := filepath.Join(sourceApp, "Contents", "MacOS", filepath.Base(currentExe))
	if _, err := os.Stat(newInnerExe); err != nil {
		macosDir := filepath.Join(sourceApp, "Contents", "MacOS")
		files, _ := os.ReadDir(macosDir)
		if len(files) > 0 {
			newInnerExe = filepath.Join(macosDir, files[0].Name())
		} else {
			return fmt.Errorf("could not find executable inside .app bundle")
		}
	}

	oldExe := currentExe + ".old"
	_ = os.Remove(oldExe)
	if err := os.Rename(currentExe, oldExe); err != nil {
		// In APFS, running binaries cannot be renamed, but unlinking (rm) works!
		_ = os.Remove(currentExe)
	}

	if err := copyFile(newInnerExe, currentExe); err != nil {
		if _, statErr := os.Stat(oldExe); statErr == nil {
			_ = os.Rename(oldExe, currentExe)
		}
		return fmt.Errorf("failed installing new binary: %w", err)
	}

	_ = os.Chmod(currentExe, 0755)
	_ = exec.Command("xattr", "-cr", currentExe).Run()
	_ = os.Remove(oldExe)
	return nil
}

const (
	maxZipEntries    = 5000
	maxZipTotalBytes = 500 * 1024 * 1024 // 500 MB maximum uncompressed size
)

func extractZipArchive(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	if len(r.File) > maxZipEntries {
		return fmt.Errorf("archive contains too many files (%d > %d)", len(r.File), maxZipEntries)
	}

	destClean := filepath.Clean(destDir) + string(os.PathSeparator)
	var totalBytesWritten int64

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(targetPath), destClean) && filepath.Clean(targetPath) != filepath.Clean(destDir) {
			continue // ZipSlip protection
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		remain := maxZipTotalBytes - totalBytesWritten
		if remain <= 0 {
			outFile.Close()
			rc.Close()
			return fmt.Errorf("archive exceeds maximum uncompressed size limit (%d MB)", maxZipTotalBytes/(1024*1024))
		}

		limitedRC := io.LimitReader(rc, remain+1)
		copied, copyErr := io.Copy(outFile, limitedRC)
		outFile.Close()
		rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if copied > remain {
			return fmt.Errorf("archive exceeds maximum uncompressed size limit (%d MB)", maxZipTotalBytes/(1024*1024))
		}
		totalBytesWritten += copied

		_ = os.Chmod(targetPath, f.Mode())
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// RestartApp launches the newly updated executable and terminates the old process
func RestartApp() error {
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}
	currentExe, _ = filepath.EvalSymlinks(currentExe)

	// Guarantee proxy cleanup before restarting process
	sysproxy.ExecuteGuaranteedCleanup()

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
