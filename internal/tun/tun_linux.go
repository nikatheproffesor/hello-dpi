//go:build linux
// +build linux

package tun

import (
	"fmt"
	"os"
)

// LinuxDevice implements Device for Linux /dev/net/tun
type LinuxDevice struct {
	file *os.File
	name string
	mtu  int
}

// OpenDevice creates a Linux TUN interface
func OpenDevice(name string) (Device, error) {
	f, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return NewMockDevice(name, 1500), nil
	}
	return &LinuxDevice{
		file: f,
		name: name,
		mtu:  1500,
	}, nil
}

func (d *LinuxDevice) Name() string {
	return d.name
}

func (d *LinuxDevice) MTU() int {
	return d.mtu
}

func (d *LinuxDevice) Read(buf []byte) (int, error) {
	if d.file == nil {
		return 0, fmt.Errorf("device closed")
	}
	return d.file.Read(buf)
}

func (d *LinuxDevice) Write(buf []byte) (int, error) {
	if d.file == nil {
		return 0, fmt.Errorf("device closed")
	}
	return d.file.Write(buf)
}

func (d *LinuxDevice) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}
