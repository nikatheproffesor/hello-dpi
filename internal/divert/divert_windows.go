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

// Start launches the WinDivert kernel engine with turkey_dnsredir parameters
func Start() error {
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

	// Standard command: -9 --dns-addr 77.88.8.8 --dns-port 1253
	cmd := exec.Command(exePath, "-9", "--dns-addr", "77.88.8.8", "--dns-port", "1253")
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
	psArgs := fmt.Sprintf(`Start-Process -FilePath "%s" -ArgumentList "-9 --dns-addr 77.88.8.8 --dns-port 1253" -Verb RunAs -WindowStyle Hidden`, exePath)
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
