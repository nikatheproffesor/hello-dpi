//go:build windows

package divert

import (
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
		targetName string
		embedName  string
	}{
		{"goodbyedpi.exe", "goodbyedpi.exe.bin"},
		{"WinDivert.dll", "WinDivert.dll.bin"},
		{"WinDivert64.sys", "WinDivert64.sys.bin"},
	}

	for _, ef := range embeddedFiles {
		targetPath := filepath.Join(targetDir, ef.targetName)
		stat, err := os.Stat(targetPath)
		if err == nil && stat.Size() > 0 {
			continue // already extracted
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

		if err := os.WriteFile(targetPath, decryptedData, 0755); err != nil {
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

// Stop terminates the WinDivert engine and cleans up services
func Stop() error {
	stateMu.Lock()
	defer stateMu.Unlock()

	if currentCmd != nil && currentCmd.Process != nil {
		_ = currentCmd.Process.Kill()
		currentCmd = nil
	}

	// Force kill any orphaned goodbyedpi.exe instances
	killCmd := exec.Command("taskkill", "/F", "/IM", "goodbyedpi.exe")
	killCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	_ = killCmd.Run()

	// Stop WinDivert kernel driver service
	stopSvc := exec.Command("net", "stop", "WinDivert")
	stopSvc.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	_ = stopSvc.Run()

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
