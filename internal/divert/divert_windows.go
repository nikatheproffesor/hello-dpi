//go:build windows

package divert

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	currentCmd *exec.Cmd
)

// ensureExtracted extracts the embedded WinDivert binaries to %LOCALAPPDATA%\HelloDPI\divert\
func ensureExtracted() (string, error) {
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = os.Getenv("APPDATA")
	}
	if localAppData == "" {
		localAppData = os.TempDir()
	}

	targetDir := filepath.Join(localAppData, "HelloDPI", "divert")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create divert directory: %w", err)
	}

	const xorKey byte = 0x5A
	embeddedFiles := []struct {
		targetName   string
		embedName    string
		expectedHash string
	}{
		{"goodbyedpi.exe", "goodbyedpi.exe.bin", "8d412b094bb9c137ff25ba9a794d1122ecc84bb776debff6c249723a13cc31cd"},
		{"WinDivert.dll", "WinDivert.dll.bin", "6110bfa44667405179c3e15e12af1b62037e447ed59b054b19042032995e6c7e"},
		{"WinDivert64.sys", "WinDivert64.sys.bin", "e69b5ba3f0cd6cfb2983e442636e7f0b342b61b15264b0328317d4559c82cf50"},
	}

	for _, ef := range embeddedFiles {
		targetPath := filepath.Join(targetDir, ef.targetName)
		if existing, err := os.ReadFile(targetPath); err == nil {
			h := sha256.Sum256(existing)
			if hex.EncodeToString(h[:]) == ef.expectedHash {
				continue // already extracted and verified intact
			}
		}

		src, err := embeddedBin.Open("bin/" + ef.embedName)
		if err != nil {
			return "", fmt.Errorf("failed to open embedded %s: %w", ef.embedName, err)
		}

		encryptedData, readErr := io.ReadAll(src)
		src.Close()
		if readErr != nil {
			return "", fmt.Errorf("failed to read embedded %s: %w", ef.embedName, readErr)
		}

		// Decrypt in-memory using XOR
		decryptedData := make([]byte, len(encryptedData))
		for i := 0; i < len(encryptedData); i++ {
			decryptedData[i] = encryptedData[i] ^ xorKey
		}

		h := sha256.Sum256(decryptedData)
		if hex.EncodeToString(h[:]) != ef.expectedHash {
			return "", fmt.Errorf("embedded asset %s integrity check failed: hash mismatch", ef.targetName)
		}

		if err := os.WriteFile(targetPath, decryptedData, 0700); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", targetPath, err)
		}
	}

	return targetDir, nil
}

func buildArgs(opts KernelOptions) []string {
	// Standard web-only DPI evasion: only intercepts ports 80/443.
	// Never intercepts UDP gaming ports (Valorant/Vivox RTP 12000-65000, Riot servers).
	args := []string{"-p", "-r", "-s", "-f", "2", "-k", "2", "-n", "-e", "2"}
	if opts.DropQUIC {
		args = append(args, "-q")
	}

	switch opts.StrategyName {
	case "wrong-seq":
		args = append(args, "--wrong-seq", "--fake-ttl", "3")
	case "wrong-checksum":
		args = append(args, "--wrong-chksum", "--fake-ttl", "3")
	case "out-of-order":
		args = append(args, "--reverse-frag", "--wrong-seq")
	case "reverse-frag":
		args = append(args, "--reverse-frag")
	case "sni":
		args = append(args, "--fragment-sni")
	case "decoy":
		args = append(args, "--fake-with-sni", "--fake-ttl", "3")
	default:
		if opts.WrongSeq {
			args = append(args, "--wrong-seq", "--fake-ttl", "3")
		} else if opts.WrongChecksum {
			args = append(args, "--wrong-chksum", "--fake-ttl", "3")
		}
	}

	// Do NOT hijack system DNS to 77.88.8.8:1253 as it breaks Valorant voice routing and matchmaking!
	if opts.DNSAddr != "" {
		port := opts.DNSPort
		if port == "" {
			port = "53"
		}
		args = append(args, "--dns-addr", opts.DNSAddr, "--dns-port", port)
	}
	return args
}

// Start launches the WinDivert kernel engine with default turkey_dnsredir parameters
func Start() error {
	return StartWithOptions(KernelOptions{
		StrategyName: "tlsrec",
		DropQUIC:     true,
	})
}

// StartStrategy launches the WinDivert engine aligned with a specific bypass strategy
func StartStrategy(stratName string) error {
	return StartWithOptions(KernelOptions{
		StrategyName: stratName,
		DropQUIC:     true,
	})
}

// StartWithOptions launches the WinDivert kernel engine with specified evasion flags
func StartWithOptions(opts KernelOptions) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	if isRunningLocked() {
		return nil
	}

	dir, err := ensureExtracted()
	if err != nil {
		return fmt.Errorf("divert engine preparation failed: %w", err)
	}

	exePath := filepath.Join(dir, "goodbyedpi.exe")
	args := buildArgs(opts)

	cmd := exec.Command(exePath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	if err := cmd.Start(); err == nil {
		currentCmd = cmd
		go func() {
			_ = cmd.Wait()
			stateMu.Lock()
			if currentCmd == cmd {
				currentCmd = nil
			}
			stateMu.Unlock()
		}()
		return nil
	}

	// If direct non-admin start fails to load driver, invoke elevated via UAC
	psArgs := fmt.Sprintf(`Start-Process -FilePath "%s" -ArgumentList "%s" -Verb RunAs -WindowStyle Hidden`,
		exePath, strings.Join(args, " "))
	cmdElevated := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psArgs)
	cmdElevated.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	if err := cmdElevated.Run(); err != nil {
		return fmt.Errorf("failed to start elevated WinDivert engine: %w", err)
	}

	return nil
}

// Stop terminates the WinDivert engine process owned by this application
func Stop() error {
	stateMu.Lock()
	defer stateMu.Unlock()

	if currentCmd != nil && currentCmd.Process != nil {
		_ = currentCmd.Process.Kill()
		_ = currentCmd.Wait()
		currentCmd = nil
	}

	return nil
}

// IsRunning checks whether the WinDivert engine is currently active
func IsRunning() bool {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return isRunningLocked()
}

func isRunningLocked() bool {
	if currentCmd != nil && currentCmd.Process != nil {
		return true
	}

	// Check if goodbyedpi.exe is present in Windows tasklist
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq goodbyedpi.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	out, err := cmd.Output()
	if err == nil && strings.Contains(string(out), "goodbyedpi.exe") {
		return true
	}

	return false
}

// GetStatus returns the current engine state
func GetStatus() Status {
	active := IsRunning()
	msg := "Çekirdek Modu kapalı"
	if active {
		msg = "Çekirdek Modu aktif (Roblox & Tüm Oyunlar Destekleniyor)"
	}
	return Status{
		Supported: true,
		Active:    active,
		Message:   msg,
	}
}
