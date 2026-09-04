//go:build linux

package sysproxy

import (
	"log"
	"os/exec"
	"strconv"
)

type linuxManager struct{}

func GetManager() Manager {
	return &linuxManager{}
}

func (m *linuxManager) Enable(host string, port int) error {
	log.Printf("[Hello DPI] Configuring Linux GNOME proxy to %s:%d", host, port)
	portStr := strconv.Itoa(port)

	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", portStr).Run()

	return nil
}

func (m *linuxManager) Disable() error {
	log.Printf("[Hello DPI] Restoring Linux GNOME proxy")
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
	return nil
}
