//go:build !windows

package divert

// Start is a no-op on non-Windows platforms
func Start() error {
	return nil
}

// Stop is a no-op on non-Windows platforms
func Stop() error {
	return nil
}

// IsRunning returns false on non-Windows platforms
func IsRunning() bool {
	return false
}

// GetStatus returns unsupported state on non-Windows platforms
func GetStatus() Status {
	return Status{
		Supported: false,
		Active:    false,
		Message:   "Çekirdek Modu yalnızca Windows için gereklidir (macOS/Linux doğrudan proxy üzerinden çalışır)",
	}
}
