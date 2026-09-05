//go:build windows

package sysproxy

import (
	"fmt"
	"log"
	"sync"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

var (
	wininetDLL            = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption = wininetDLL.NewProc("InternetSetOptionW")
)

const (
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
	bypassList                    = "<local>;*.local;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;*.gsb.gov.tr;*.kyk.gov.tr;captive.apple.com;connectivitycheck.gstatic.com;msftconnecttest.com"
)

type windowsManager struct {
	mu           sync.Mutex
	hadProxy     bool
	prevServer   string
	prevOverride string
}

func GetManager() Manager {
	return &windowsManager{}
}

func notifyWinINet() {
	_, _, _ = procInternetSetOption.Call(0, uintptr(internetOptionSettingsChanged), 0, 0)
	_, _, _ = procInternetSetOption.Call(0, uintptr(internetOptionRefresh), 0, 0)
}

func (m *windowsManager) Enable(host string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open Internet Settings registry key: %w", err)
	}
	defer key.Close()

	// Backup existing proxy settings
	if val, _, err := key.GetIntegerValue("ProxyEnable"); err == nil {
		m.hadProxy = (val == 1)
	}
	if val, _, err := key.GetStringValue("ProxyServer"); err == nil {
		m.prevServer = val
	}
	if val, _, err := key.GetStringValue("ProxyOverride"); err == nil {
		m.prevOverride = val
	}

	proxyAddr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[Hello DPI] Configuring Windows Internet Settings proxy to %s (instant Win32 Registry)", proxyAddr)

	_ = key.SetDWordValue("ProxyEnable", 1)
	_ = key.SetStringValue("ProxyServer", proxyAddr)
	_ = key.SetStringValue("ProxyOverride", bypassList)

	notifyWinINet()
	return nil
}

func (m *windowsManager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[Hello DPI] Restoring Windows Internet Settings proxy")
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open Internet Settings registry key: %w", err)
	}
	defer key.Close()

	if m.hadProxy && m.prevServer != "" {
		_ = key.SetDWordValue("ProxyEnable", 1)
		_ = key.SetStringValue("ProxyServer", m.prevServer)
		if m.prevOverride != "" {
			_ = key.SetStringValue("ProxyOverride", m.prevOverride)
		}
	} else {
		_ = key.SetDWordValue("ProxyEnable", 0)
	}

	notifyWinINet()
	return nil
}
