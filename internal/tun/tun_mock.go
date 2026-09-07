package tun

import (
	"sync"
)

// MockDevice provides an in-memory synthetic TUN interface for testing
type MockDevice struct {
	mu      sync.Mutex
	name    string
	mtu     int
	inPipe  chan []byte
	outPipe chan []byte
	closed  bool
}

// NewMockDevice creates an in-memory virtual device
func NewMockDevice(name string, mtu int) *MockDevice {
	if mtu <= 0 {
		mtu = 1500
	}
	return &MockDevice{
		name:    name,
		mtu:     mtu,
		inPipe:  make(chan []byte, 100),
		outPipe: make(chan []byte, 100),
	}
}

func (m *MockDevice) Name() string {
	return m.name
}

func (m *MockDevice) MTU() int {
	return m.mtu
}

func (m *MockDevice) Read(buf []byte) (int, error) {
	pkt, ok := <-m.inPipe
	if !ok {
		return 0, ErrDeviceClosed
	}
	copy(buf, pkt)
	return len(pkt), nil
}

func (m *MockDevice) Write(buf []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, ErrDeviceClosed
	}
	pkt := make([]byte, len(buf))
	copy(pkt, buf)
	select {
	case m.outPipe <- pkt:
	default:
	}
	return len(buf), nil
}

// InjectPacket feeds an inbound IP packet into the device (simulating incoming network traffic)
func (m *MockDevice) InjectPacket(pkt []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	select {
	case m.inPipe <- pkt:
	default:
	}
}

func (m *MockDevice) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.inPipe)
	}
	return nil
}
