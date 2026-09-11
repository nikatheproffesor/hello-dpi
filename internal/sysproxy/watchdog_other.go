//go:build !darwin && !windows

package sysproxy

func startWatchdog(host string, port int) {}
func stopWatchdog()                       {}
