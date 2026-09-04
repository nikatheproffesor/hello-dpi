//go:build windows

package sysproxy

import (
	"fmt"
	"log"
	"os/exec"
)

type windowsManager struct{}

func GetManager() Manager {
	return &windowsManager{}
}

func (m *windowsManager) Enable(host string, port int) error {
	proxyAddr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("[Hello DPI] Configuring Windows Internet Settings proxy to %s", proxyAddr)

	regKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

	_ = exec.Command("reg", "add", regKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f").Run()
	_ = exec.Command("reg", "add", regKey, "/v", "ProxyServer", "/t", "REG_SZ", "/d", proxyAddr, "/f").Run()

	return nil
}

func (m *windowsManager) Disable() error {
	log.Printf("[Hello DPI] Restoring Windows Internet Settings proxy")
	regKey := `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`
	_ = exec.Command("reg", "add", regKey, "/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f").Run()
	return nil
}
