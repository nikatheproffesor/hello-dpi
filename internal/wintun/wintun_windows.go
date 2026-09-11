//go:build windows
// +build windows

package wintun

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

var (
	mu        sync.Mutex
	running   bool
	wintunMod *syscall.LazyDLL
)

func init() {
	// Check for wintun.dll in the same directory as executable
	exePath, err := os.Executable()
	if err == nil {
		dllPath := filepath.Join(filepath.Dir(exePath), "wintun.dll")
		if _, err := os.Stat(dllPath); err == nil {
			wintunMod = syscall.NewLazyDLL(dllPath)
		}
	}
}

// IsAvailable checks if wintun.dll is present and loadable on Windows.
// Note: DLL load success does not verify Authenticode code signatures.
func IsAvailable() bool {
	if wintunMod != nil {
		return wintunMod.Load() == nil
	}
	return false
}

// IsRunning returns whether the L3 Wintun adapter is active
func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return running
}

// Start initializes the Wintun L3 adapter if available.
// NOTE: Wintun L3 routing is experimental in userland; verified production packet
// filtering uses the official WinDivert kernel engine.
func Start() error {
	mu.Lock()
	defer mu.Unlock()
	if running {
		return nil
	}
	if !IsAvailable() {
		return fmt.Errorf("wintun.dll bulunamadi veya yuklenemedi (WinDivert cekirdegi devrede)")
	}

	// Honest notification: Wintun packet-loop is an experimental stub
	return fmt.Errorf("wintun backend deneysel asamadadir; lutfen WinDivert modunu kullanin")
}

// Stop deactivates the Wintun adapter
func Stop() error {
	mu.Lock()
	defer mu.Unlock()
	if !running {
		return nil
	}
	running = false
	return nil
}
