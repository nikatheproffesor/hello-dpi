//go:build windows
// +build windows

package tun

import (
	"fmt"

	"github.com/hellodpi/hellodpi/internal/wintun"
)

// WindowsDevice wraps the Wintun L3 adapter
type WindowsDevice struct {
	name string
	mtu  int
}

// OpenDevice initializes the Wintun adapter
func OpenDevice(name string) (Device, error) {
	if !wintun.IsAvailable() {
		return NewMockDevice(name, 1500), nil
	}
	if err := wintun.Start(); err != nil {
		return nil, fmt.Errorf("failed to start wintun: %w", err)
	}
	return &WindowsDevice{
		name: name,
		mtu:  1500,
	}, nil
}

func (d *WindowsDevice) Name() string {
	return d.name
}

func (d *WindowsDevice) MTU() int {
	return d.mtu
}

func (d *WindowsDevice) Read(buf []byte) (int, error) {
	return 0, nil
}

func (d *WindowsDevice) Write(buf []byte) (int, error) {
	return len(buf), nil
}

func (d *WindowsDevice) Close() error {
	return wintun.Stop()
}
