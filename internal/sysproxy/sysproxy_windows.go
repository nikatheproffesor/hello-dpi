//go:build windows

package sysproxy

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

var (
	wininetDLL            = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption = wininetDLL.NewProc("InternetSetOptionW")
)

const (
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
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

	regKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

	// Backup existing proxy settings
	if out, err := exec.Command("reg", "query", regKey, "/v", "ProxyEnable").Output(); err == nil {
		m.hadProxy = strings.Contains(string(out), "0x1")
	}
	if out, err := exec.Command("reg", "query", regKey, "/v", "ProxyServer").Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) >= 3 {
			m.prevServer = fields[len(fields)-1]
		}
	}

	proxyAddr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[Hello DPI] Configuring Windows Internet Settings proxy to %s", proxyAddr)

	_ = exec.Command("reg", "add", regKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run()
	_ = exec.Command("reg", "add", regKey, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f").Run()

	notifyWinINet()
	return nil
}

func (m *windowsManager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[Hello DPI] Restoring Windows Internet Settings proxy")
	regKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

	if m.hadProxy && m.prevServer != "" {
		_ = exec.Command("reg", "add", regKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run()
		_ = exec.Command("reg", "add", regKey, "/v", "ProxyServer", "/t", "REG_SZ", "/d", m.prevServer, "/f").Run()
	} else {
		_ = exec.Command("reg", "add", regKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	}

	notifyWinINet()
	return nil
}
