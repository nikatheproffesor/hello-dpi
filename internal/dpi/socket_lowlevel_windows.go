//go:build windows

package dpi

import "syscall"

func setTTLNative(fd uintptr, ttl int) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}

func setMSSNative(fd uintptr, mss int) error {
	// Winsock does not support SO_MAXSEG on established TCP sockets; fallback gracefully
	return nil
}
