//go:build windows

package sysproxy

import (
	"fmt"
	"log"
	"os/exec"
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
	bypassList                    = "<-loopback>;127.0.0.1;localhost;<local>;*.local;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;*.gsb.gov.tr;*.kyk.gov.tr;captive.apple.com;connectivitycheck.gstatic.com;msftconnecttest.com"
)

type windowsManager struct {
	mu             sync.Mutex
	backedUp       bool
	hadProxy       bool
	prevServer     string
	prevOverride   string
	configuredAddr string
}

var winMgr = &windowsManager{}

func GetManager() Manager {
	return winMgr
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

	// Backup existing proxy settings once per lifecycle session
	if !m.backedUp {
		if val, _, err := key.GetIntegerValue("ProxyEnable"); err == nil {
			m.hadProxy = (val == 1)
		}
		if val, _, err := key.GetStringValue("ProxyServer"); err == nil {
			m.prevServer = val
		}
		if val, _, err := key.GetStringValue("ProxyOverride"); err == nil {
			m.prevOverride = val
		}
		m.backedUp = true
	}

	proxyAddr := fmt.Sprintf("%s:%d", host, port)
	m.configuredAddr = proxyAddr
	log.Printf("[Hello DPI] Configuring Windows Internet Settings proxy to %s (instant Win32 Registry)", proxyAddr)

	if err := key.SetDWordValue("ProxyEnable", 1); err != nil {
		return fmt.Errorf("failed to set ProxyEnable registry value: %w", err)
	}
	if err := key.SetStringValue("ProxyServer", proxyAddr); err != nil {
		return fmt.Errorf("failed to set ProxyServer registry value: %w", err)
	}
	if err := key.SetStringValue("ProxyOverride", bypassList); err != nil {
		return fmt.Errorf("failed to set ProxyOverride registry value: %w", err)
	}

	notifyWinINet()

	// Flush Windows DNS cache
	go func() {
		_ = exec.Command("ipconfig", "/flushdns").Run()
	}()

	return nil
}

func (m *windowsManager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[Hello DPI] Restoring Windows Internet Settings proxy")
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open Internet Settings registry key: %w", err)
	}
	defer key.Close()

	if m.backedUp && m.hadProxy && m.prevServer != "" && m.prevServer != m.configuredAddr {
		// Restore previous custom/corporate proxy settings
		_ = key.SetDWordValue("ProxyEnable", 1)
		_ = key.SetStringValue("ProxyServer", m.prevServer)
		if m.prevOverride != "" {
			_ = key.SetStringValue("ProxyOverride", m.prevOverride)
		}
	} else {
		// Clean direct restoration
		_ = key.SetDWordValue("ProxyEnable", 0)
		if currentServer, _, err := key.GetStringValue("ProxyServer"); err == nil && currentServer == m.configuredAddr {
			_ = key.DeleteValue("ProxyServer")
			_ = key.DeleteValue("ProxyOverride")
		}
	}

	m.backedUp = false
	m.hadProxy = false
	m.prevServer = ""
	m.prevOverride = ""
	m.configuredAddr = ""

	notifyWinINet()

	// Clean up user environment variables if they were set
	if envKey, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE); err == nil {
		_ = envKey.DeleteValue("HTTP_PROXY")
		_ = envKey.DeleteValue("HTTPS_PROXY")
		_ = envKey.DeleteValue("ALL_PROXY")
		envKey.Close()
	}

	go func() {
		_ = exec.Command("ipconfig", "/flushdns").Run()
	}()

	return nil
}
