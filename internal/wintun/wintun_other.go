//go:build !windows
// +build !windows

package wintun

import (
	"errors"
)

// IsAvailable returns false on non-Windows platforms (uses sysproxy / utun instead)
func IsAvailable() bool {
	return false
}

// IsRunning returns whether Wintun is active
func IsRunning() bool {
	return false
}

// Start is a no-op on non-Windows
func Start() error {
	return errors.New("wintun is only supported on Windows")
}

// Stop is a no-op on non-Windows
func Stop() error {
	return nil
}
