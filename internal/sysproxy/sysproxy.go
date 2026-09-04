package sysproxy

import (
	"log"
)

// Manager provides system-level proxy configuration
type Manager interface {
	Enable(host string, port int) error
	Disable() error
}

// SetSystemProxy enables system proxy on the current platform
func SetSystemProxy(host string, port int) error {
	m := GetManager()
	if m == nil {
		log.Printf("[Hello DPI] System proxy auto-configuration is not supported on this platform.")
		return nil
	}
	return m.Enable(host, port)
}

// ClearSystemProxy restores system proxy settings on the current platform
func ClearSystemProxy() error {
	m := GetManager()
	if m == nil {
		return nil
	}
	return m.Disable()
}
