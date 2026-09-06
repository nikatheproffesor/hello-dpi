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

	files := []string{"goodbyedpi.exe", "WinDivert.dll", "WinDivert64.sys"}
	for _, f := range files {
		targetPath := filepath.Join(targetDir, f)
		stat, err := os.Stat(targetPath)
		if err == nil && stat.Size() > 0 {
			continue // already extracted
		}

		src, err := embeddedBin.Open("bin/" + f)
		if err != nil {
			return "", fmt.Errorf("failed to open embedded %s: %w", f, err)
		}

		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			src.Close()
			return "", fmt.Errorf("failed to write %s: %w", targetPath, err)
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return "", fmt.Errorf("failed to copy %s: %w", f, copyErr)
		}
	}

	return targetDir, nil
}

func buildArgs(opts KernelOptions) []string {
	args := []string{"-9"}
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

	dnsAddr := opts.DNSAddr
	if dnsAddr == "" {
		dnsAddr = "77.88.8.8"
	}
	dnsPort := opts.DNSPort
	if dnsPort == "" {
		dnsPort = "1253"
	}
	args = append(args, "--dns-addr", dnsAddr, "--dns-port", dnsPort)
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
