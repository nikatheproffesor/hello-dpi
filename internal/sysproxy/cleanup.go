package sysproxy

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hellodpi/hellodpi/internal/divert"
)

var (
	cleanupOnce sync.Once
	cleanedUp   bool
	cleanupMu   sync.Mutex
)

// RegisterExitCleanup guarantees that on OS shutdown, Ctrl+C, SIGTERM, or SIGHUP,
// the system proxy and kernel divert drivers are cleanly disabled.
func RegisterExitCleanup() {
	cleanupOnce.Do(func() {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

		go func() {
			sig := <-sigCh
			log.Printf("[Hello DPI] Termination signal received (%v). Restoring system proxy and stopping kernel driver...", sig)
			ExecuteGuaranteedCleanup()
			os.Exit(0)
		}()
	})
}

// ExecuteGuaranteedCleanup performs an idempotent, immediate restoration of network state
func ExecuteGuaranteedCleanup() {
	cleanupMu.Lock()
	defer cleanupMu.Unlock()

	if cleanedUp {
		return
	}
	cleanedUp = true

	if IsSystemProxyActive() {
		_ = ClearSystemProxy()
		log.Printf("[Hello DPI] Exit cleanup completed: system proxy restored.")
	}
	_ = divert.Stop()
}

// RecoverAndClear can be deferred in main() or panicking goroutines to prevent leaving
// system proxy pointing to a terminated process.
func RecoverAndClear() {
	if r := recover(); r != nil {
		log.Printf("[Hello DPI FATAL] Panic caught: %v. Executing emergency system proxy restoration...", r)
		ExecuteGuaranteedCleanup()
		panic(r) // Re-throw panic after network restoration
	}
}
