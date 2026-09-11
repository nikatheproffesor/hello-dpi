//go:build darwin

package sysproxy

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type proxySetting struct {
	enabled bool
	server  string
	port    string
}

type serviceBackup struct {
	httpProxy     proxySetting
	httpsProxy    proxySetting
	socksProxy    proxySetting
	bypassDomains []string
	hadBypass     bool
}

type darwinManager struct {
	mu             sync.Mutex
	activeServices []string
	backups        map[string]serviceBackup
	configuredHost string
	configuredPort string
}

var darwinMgr = &darwinManager{
	backups: make(map[string]serviceBackup),
}

func GetManager() Manager {
	return darwinMgr
}

func (m *darwinManager) getServices() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return []string{"Wi-Fi", "Ethernet"}
	}

	lines := strings.Split(string(out), "\n")
	var services []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "An asterisk") {
			continue
		}
		services = append(services, l)
	}
	if len(services) == 0 {
		return []string{"Wi-Fi", "Ethernet"}
	}
	return services
}

func parseProxyOutput(out []byte) proxySetting {
	s := proxySetting{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch strings.ToLower(key) {
		case "enabled":
			s.enabled = (strings.ToLower(val) == "yes")
		case "server":
			s.server = val
		case "port":
			s.port = val
		}
	}
	return s
}

func (m *darwinManager) backupService(service string) serviceBackup {
	sb := serviceBackup{}

	if out, err := exec.Command("networksetup", "-getwebproxy", service).Output(); err == nil {
		sb.httpProxy = parseProxyOutput(out)
	}
	if out, err := exec.Command("networksetup", "-getsecurewebproxy", service).Output(); err == nil {
		sb.httpsProxy = parseProxyOutput(out)
	}
	if out, err := exec.Command("networksetup", "-getsocksfirewallproxy", service).Output(); err == nil {
		sb.socksProxy = parseProxyOutput(out)
	}

	// Backup existing proxy bypass domains
	if out, err := exec.Command("networksetup", "-getproxybypassdomains", service).Output(); err == nil {
		outStr := strings.TrimSpace(string(out))
		if outStr != "" && !strings.Contains(outStr, "aren't any") && !strings.Contains(outStr, "No bypass") {
			lines := strings.Fields(outStr)
			if len(lines) > 0 {
				sb.bypassDomains = lines
				sb.hadBypass = true
			}
		}
	}

	return sb
}

func (m *darwinManager) Enable(host string, port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeServices = m.getServices()
	portStr := strconv.Itoa(port)
	m.configuredHost = host
	m.configuredPort = portStr

	for _, s := range m.activeServices {
		// Only backup if we haven't already backed up this service in this session
		if _, exists := m.backups[s]; !exists {
			m.backups[s] = m.backupService(s)
		}

		log.Printf("[Hello DPI] Configuring system proxy for macOS network service: %s", s)
		_ = exec.Command("networksetup", "-setwebproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setsecurewebproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setproxybypassdomains", s, "127.0.0.1", "localhost", "*.local", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "*.gsb.gov.tr", "*.kyk.gov.tr", "captive.apple.com", "connectivitycheck.gstatic.com", "msftconnecttest.com").Run()
	}

	return nil
}

func (m *darwinManager) Disable() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	services := m.activeServices
	if len(services) == 0 {
		services = m.getServices()
	}

	for _, s := range services {
		log.Printf("[Hello DPI] Restoring previous proxy settings for service: %s", s)

		backup, hasBackup := m.backups[s]
		if hasBackup {
			isOurHTTP := backup.httpProxy.server == m.configuredHost && backup.httpProxy.port == m.configuredPort
			isOurHTTPS := backup.httpsProxy.server == m.configuredHost && backup.httpsProxy.port == m.configuredPort
			isOurSOCKS := backup.socksProxy.server == m.configuredHost && backup.socksProxy.port == m.configuredPort

			// Restore HTTP (preserve another proxy if it wasn't ours)
			if backup.httpProxy.enabled && backup.httpProxy.server != "" && !isOurHTTP {
				_ = exec.Command("networksetup", "-setwebproxy", s, backup.httpProxy.server, backup.httpProxy.port).Run()
				_ = exec.Command("networksetup", "-setwebproxystate", s, "on").Run()
			} else {
				_ = exec.Command("networksetup", "-setwebproxystate", s, "off").Run()
			}

			// Restore HTTPS
			if backup.httpsProxy.enabled && backup.httpsProxy.server != "" && !isOurHTTPS {
				_ = exec.Command("networksetup", "-setsecurewebproxy", s, backup.httpsProxy.server, backup.httpsProxy.port).Run()
				_ = exec.Command("networksetup", "-setsecurewebproxystate", s, "on").Run()
			} else {
				_ = exec.Command("networksetup", "-setsecurewebproxystate", s, "off").Run()
			}

			// Restore SOCKS
			if backup.socksProxy.enabled && backup.socksProxy.server != "" && !isOurSOCKS {
				_ = exec.Command("networksetup", "-setsocksfirewallproxy", s, backup.socksProxy.server, backup.socksProxy.port).Run()
				_ = exec.Command("networksetup", "-setsocksfirewallproxystate", s, "on").Run()
			} else {
				_ = exec.Command("networksetup", "-setsocksfirewallproxystate", s, "off").Run()
			}

			// Restore proxy bypass domains
			if backup.hadBypass && len(backup.bypassDomains) > 0 {
				args := append([]string{"-setproxybypassdomains", s}, backup.bypassDomains...)
				_ = exec.Command("networksetup", args...).Run()
			} else {
				_ = exec.Command("networksetup", "-setproxybypassdomains", s, "empty").Run()
			}
		} else {
			// Blanket off fallback
			_ = exec.Command("networksetup", "-setwebproxystate", s, "off").Run()
			_ = exec.Command("networksetup", "-setsecurewebproxystate", s, "off").Run()
			_ = exec.Command("networksetup", "-setsocksfirewallproxystate", s, "off").Run()
			_ = exec.Command("networksetup", "-setproxybypassdomains", s, "empty").Run()
		}
	}

	// Always clean up any environment variables from macOS user session to prevent broken IDE/shell connections
	for _, envVar := range []string{"http_proxy", "https_proxy", "all_proxy", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
		_ = exec.Command("launchctl", "unsetenv", envVar).Run()
	}

	m.backups = make(map[string]serviceBackup)
	m.configuredHost = ""
	m.configuredPort = ""
	return nil
}
