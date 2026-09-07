//go:build windows

package sysproxy

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

var watchdogCmd *exec.Cmd

func startWatchdog(host string, port int) {
	stopWatchdog()

	pid := os.Getpid()
	// Detached PowerShell guardian running hidden with no window.
	// Continuously checks if Hello DPI PID exists. The moment PID terminates
	// (antivirus quarantine, crash, Task Manager kill, or uninstaller),
	// it immediately turns off ProxyEnable, removes ProxyServer, and resets WinHTTP.
	psScript := fmt.Sprintf(`
$pidToWatch = %d
while (Get-Process -Id $pidToWatch -ErrorAction SilentlyContinue) {
    Start-Sleep -Seconds 2
}
$proxy = (Get-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyServer -ErrorAction SilentlyContinue).ProxyServer
if ($proxy -like '*127.0.0.1*' -or $proxy -like '*:%d*') {
    Set-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyEnable -Value 0
    Remove-ItemProperty -Path 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings' -Name ProxyServer -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path 'HKCU:\Environment' -Name HTTP_PROXY -ErrorAction SilentlyContinue
    Remove-ItemProperty -Path 'HKCU:\Environment' -Name HTTPS_PROXY -ErrorAction SilentlyContinue
    netsh winhttp reset proxy
    ipconfig /flushdns
}
`, pid, port)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000200, // CREATE_NO_WINDOW | CREATE_NEW_PROCESS_GROUP
	}

	if err := cmd.Start(); err == nil {
		watchdogCmd = cmd
	}
}

func stopWatchdog() {
	if watchdogCmd != nil && watchdogCmd.Process != nil {
		_ = watchdogCmd.Process.Kill()
		watchdogCmd = nil
	}
}
