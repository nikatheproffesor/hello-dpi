package hellocore

import (
	"sync"

	"github.com/hellodpi/hellodpi/internal/divert"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
)

// PlatformAdapter abstracts platform-specific operating system hooks
// (such as WinDivert on Windows, VpnService on Android, NetworkExtension on iOS, and system proxy)
type PlatformAdapter interface {
	Name() string
	OnStart() error
	OnStop() error
	IsActive() bool
}

// DesktopAdapter manages desktop OS system proxy and selective kernel divert
type DesktopAdapter struct {
	mu           sync.Mutex
	active       bool
	enableKernel bool
}

func NewDesktopAdapter(enableKernel bool) *DesktopAdapter {
	return &DesktopAdapter{enableKernel: enableKernel}
}

func (d *DesktopAdapter) Name() string {
	return "desktop"
}

func (d *DesktopAdapter) OnStart() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
	if d.enableKernel {
		_ = divert.Start()
	}
	d.active = true
	return nil
}

func (d *DesktopAdapter) OnStop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	_ = sysproxy.ClearSystemProxy()
	if d.enableKernel {
		_ = divert.Stop()
	}
	d.active = false
	return nil
}

func (d *DesktopAdapter) IsActive() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active
}

// AndroidVPNAdapter handles Android VpnService hooks in headless mode
type AndroidVPNAdapter struct {
	mu     sync.Mutex
	active bool
}

func NewAndroidVPNAdapter() *AndroidVPNAdapter {
	return &AndroidVPNAdapter{}
}

func (a *AndroidVPNAdapter) Name() string {
	return "android-vpn"
}

func (a *AndroidVPNAdapter) OnStart() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.active = true
	return nil
}

func (a *AndroidVPNAdapter) OnStop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.active = false
	return nil
}

func (a *AndroidVPNAdapter) IsActive() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.active
}

// IOSNetworkExtensionAdapter handles Apple Network Extension NEPacketTunnelProvider hooks
type IOSNetworkExtensionAdapter struct {
	mu     sync.Mutex
	active bool
}

func NewIOSNetworkExtensionAdapter() *IOSNetworkExtensionAdapter {
	return &IOSNetworkExtensionAdapter{}
}

func (i *IOSNetworkExtensionAdapter) Name() string {
	return "ios-network-extension"
}

func (i *IOSNetworkExtensionAdapter) OnStart() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.active = true
	return nil
}

func (i *IOSNetworkExtensionAdapter) OnStop() error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.active = false
	return nil
}

func (i *IOSNetworkExtensionAdapter) IsActive() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.active
}
