//go:build !windows

package dpi

import "syscall"

func setTTLNative(fd uintptr, ttl int) error {
	return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}

func setMSSNative(fd uintptr, mss int) error {
	return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_MAXSEG, mss)
}
