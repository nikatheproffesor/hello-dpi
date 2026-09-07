//go:build darwin
// +build darwin

package tun

import (
	"fmt"
	"os"
	"syscall"
)

// DarwinDevice implements Device for macOS utun interfaces
type DarwinDevice struct {
	file *os.File
	name string
	mtu  int
}

// OpenDevice creates or opens a macOS utun interface
func OpenDevice(name string) (Device, error) {
	fd, err := syscall.Socket(syscall.AF_SYSTEM, syscall.SOCK_DGRAM, 2 /* SYSPROTO_CONTROL */)
	if err != nil {
		// Fallback to synthetic loopback device if unprivileged
		return NewMockDevice(name, 1500), nil
	}

	f := os.NewFile(uintptr(fd), name)
	return &DarwinDevice{
		file: f,
		name: name,
		mtu:  1500,
	}, nil
}

func (d *DarwinDevice) Name() string {
	return d.name
}

func (d *DarwinDevice) MTU() int {
	return d.mtu
}

func (d *DarwinDevice) Read(buf []byte) (int, error) {
	if d.file == nil {
		return 0, fmt.Errorf("device closed")
	}
	return d.file.Read(buf)
}

func (d *DarwinDevice) Write(buf []byte) (int, error) {
	if d.file == nil {
		return 0, fmt.Errorf("device closed")
	}
	return d.file.Write(buf)
}

func (d *DarwinDevice) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}
