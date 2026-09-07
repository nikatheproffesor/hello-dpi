package sysproxy

import (
	"log"
)

// Manager provides system-level proxy configuration
type Manager interface {
	Enable(host string, port int) error
	Disable() error
}

// SetSystemProxy enables system proxy on the current platform and arms the self-healing watchdog
func SetSystemProxy(host string, port int) error {
	m := GetManager()
	if m == nil {
		log.Printf("[Hello DPI] System proxy auto-configuration is not supported on this platform.")
		return nil
	}
	err := m.Enable(host, port)
	if err == nil {
		StartWatchdog(host, port)
	}
	return err
}

// ClearSystemProxy restores system proxy settings on the current platform and disarms the watchdog
func ClearSystemProxy() error {
	StopWatchdog()
	m := GetManager()
	if m == nil {
		return nil
	}
	return m.Disable()
}
