//go:build linux

package sysproxy

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
)

type linuxManager struct{}

func GetManager() Manager {
	return &linuxManager{}
}

func (m *linuxManager) Enable(host string, port int) error {
	log.Printf("[Hello DPI] Configuring Linux system proxy to %s:%d", host, port)
	portStr := strconv.Itoa(port)
	proxyURL := fmt.Sprintf("http://%s:%d", host, port)

	// 1. GNOME Desktop
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", portStr).Run()

	// 2. KDE Plasma Desktop (KDE 5 & 6)
	for _, kcmd := range []string{"kwriteconfig6", "kwriteconfig5"} {
		if _, err := exec.LookPath(kcmd); err == nil {
			_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "1").Run()
			_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpProxy", proxyURL).Run()
			_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpsProxy", proxyURL).Run()
		}
	}

	return nil
}

func (m *linuxManager) Disable() error {
	log.Printf("[Hello DPI] Restoring Linux system proxy")

	// 1. GNOME
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()

	// 2. KDE
	for _, kcmd := range []string{"kwriteconfig6", "kwriteconfig5"} {
		if _, err := exec.LookPath(kcmd); err == nil {
			_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0").Run()
		}
	}

	return nil
}
