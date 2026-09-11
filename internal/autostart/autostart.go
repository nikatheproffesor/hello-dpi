package autostart

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const AppName = "HelloDPI"

// IsEnabled checks if launch on boot is configured for the current user
func IsEnabled() bool {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.hellodpi.tray.plist")
		_, err = os.Stat(plistPath)
		return err == nil

	case "windows":
		cmd := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", AppName)
		return cmd.Run() == nil

	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		desktopPath := filepath.Join(home, ".config", "autostart", "hellodpi.desktop")
		_, err = os.Stat(desktopPath)
		return err == nil
	}
	return false
}

// Enable registers the application to launch automatically on login
func Enable() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		agentsDir := filepath.Join(home, "Library", "LaunchAgents")
		_ = os.MkdirAll(agentsDir, 0755)
		plistPath := filepath.Join(agentsDir, "com.hellodpi.tray.plist")

		escapedPath := html.EscapeString(execPath)
		// If running from an .app bundle, use the bundle or executable
		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.hellodpi.tray</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, escapedPath)
		return os.WriteFile(plistPath, []byte(plistContent), 0644)

	case "windows":
		cmd := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", AppName, "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s"`, execPath), "/f")
		return cmd.Run()

	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		autostartDir := filepath.Join(home, ".config", "autostart")
		_ = os.MkdirAll(autostartDir, 0755)
		desktopPath := filepath.Join(autostartDir, "hellodpi.desktop")
		escapedExec := strings.ReplaceAll(execPath, `"`, `\"`)
		content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Hello DPI
Exec="%s"
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
`, escapedExec)
		return os.WriteFile(desktopPath, []byte(content), 0644)
	}
	return nil
}

// Disable removes launch on boot configuration
func Disable() error {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.hellodpi.tray.plist")
		_ = os.Remove(plistPath)
		return nil

	case "windows":
		cmd := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", AppName, "/f")
		return cmd.Run()

	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		desktopPath := filepath.Join(home, ".config", "autostart", "hellodpi.desktop")
		_ = os.Remove(desktopPath)
		return nil
	}
	return nil
}
