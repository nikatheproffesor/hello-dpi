//go:build linux

package sysproxy

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type linuxManager struct {
	mu                  sync.Mutex
	backedUp            bool
	prevGnomeMode       string
	prevGnomeHTTPHost   string
	prevGnomeHTTPPort   string
	prevGnomeHTTPSHost  string
	prevGnomeHTTPSPort  string
	prevGnomeIgnoreList string
	prevKDEProxyType    string
	prevKDEHTTPProxy    string
}

var lnxMgr = &linuxManager{}

func GetManager() Manager {
	return lnxMgr
}

func (m *linuxManager) Enable(host string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[Hello DPI] Configuring Linux system proxy to %s:%d", host, port)
	portStr := strconv.Itoa(port)
	proxyURL := fmt.Sprintf("http://%s:%d", host, port)

	// Backup GNOME settings if available
	if !m.backedUp {
		if _, err := exec.LookPath("gsettings"); err == nil {
			if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode").Output(); err == nil {
				m.prevGnomeMode = strings.TrimSpace(string(out))
			}
			if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "host").Output(); err == nil {
				m.prevGnomeHTTPHost = strings.TrimSpace(string(out))
			}
			if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "port").Output(); err == nil {
				m.prevGnomeHTTPPort = strings.TrimSpace(string(out))
			}
			if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.https", "host").Output(); err == nil {
				m.prevGnomeHTTPSHost = strings.TrimSpace(string(out))
			}
			if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy.https", "port").Output(); err == nil {
				m.prevGnomeHTTPSPort = strings.TrimSpace(string(out))
			}
			if out, err := exec.Command("gsettings", "get", "org.gnome.system.proxy", "ignore-hosts").Output(); err == nil {
				m.prevGnomeIgnoreList = strings.TrimSpace(string(out))
			}
		}

		for _, kread := range []string{"kreadconfig6", "kreadconfig5"} {
			if _, err := exec.LookPath(kread); err == nil {
				if out, err := exec.Command(kread, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType").Output(); err == nil {
					m.prevKDEProxyType = strings.TrimSpace(string(out))
				}
				if out, err := exec.Command(kread, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpProxy").Output(); err == nil {
					m.prevKDEHTTPProxy = strings.TrimSpace(string(out))
				}
				break
			}
		}
		m.backedUp = true
	}

	// 1. GNOME Desktop
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "manual").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.https", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "host", host).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.socks", "port", portStr).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "ignore-hosts", "['localhost', '127.0.0.0/8', '::1', '10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16', '*.local', '*.gsb.gov.tr', '*.kyk.gov.tr', 'captive.apple.com', 'connectivitycheck.gstatic.com', 'msftconnecttest.com']").Run()

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
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[Hello DPI] Restoring Linux system proxy")

	// 1. GNOME restoration
	if m.backedUp && m.prevGnomeMode != "" && m.prevGnomeMode != "'manual'" && m.prevGnomeMode != "manual" {
		mode := strings.Trim(m.prevGnomeMode, "'\"")
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", mode).Run()
		if m.prevGnomeHTTPHost != "" {
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "host", strings.Trim(m.prevGnomeHTTPHost, "'\"")).Run()
		}
		if m.prevGnomeHTTPPort != "" {
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy.http", "port", strings.Trim(m.prevGnomeHTTPPort, "'\"")).Run()
		}
		if m.prevGnomeIgnoreList != "" {
			_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "ignore-hosts", m.prevGnomeIgnoreList).Run()
		}
	} else {
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "none").Run()
	}

	// 2. KDE restoration
	for _, kcmd := range []string{"kwriteconfig6", "kwriteconfig5"} {
		if _, err := exec.LookPath(kcmd); err == nil {
			if m.backedUp && m.prevKDEProxyType != "" && m.prevKDEProxyType != "1" {
				_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", m.prevKDEProxyType).Run()
				if m.prevKDEHTTPProxy != "" {
					_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "httpProxy", m.prevKDEHTTPProxy).Run()
				}
			} else {
				_ = exec.Command(kcmd, "--file", "kioslaverc", "--group", "Proxy Settings", "--key", "ProxyType", "0").Run()
			}
		}
	}

	m.backedUp = false
	m.prevGnomeMode = ""
	m.prevGnomeHTTPHost = ""
	m.prevGnomeHTTPPort = ""
	m.prevGnomeIgnoreList = ""
	m.prevKDEProxyType = ""
	m.prevKDEHTTPProxy = ""

	return nil
}
