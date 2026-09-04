//go:build darwin

package sysproxy

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type darwinManager struct {
	activeServices []string
}

func GetManager() Manager {
	return &darwinManager{}
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
		// Filter out asterisk lines
		services = append(services, l)
	}
	if len(services) == 0 {
		return []string{"Wi-Fi", "Ethernet"}
	}
	return services
}

func (m *darwinManager) Enable(host string, port int) error {
	m.activeServices = m.getServices()
	portStr := strconv.Itoa(port)

	for _, s := range m.activeServices {
		log.Printf("[Hello DPI] Configuring system proxy for macOS network service: %s", s)
		_ = exec.Command("networksetup", "-setwebproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setsecurewebproxy", s, host, portStr).Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxy", s, host, portStr).Run()
	}
	return nil
}

func (m *darwinManager) Disable() error {
	services := m.activeServices
	if len(services) == 0 {
		services = m.getServices()
	}

	for _, s := range services {
		log.Printf("[Hello DPI] Restoring system proxy for macOS network service: %s", s)
		_ = exec.Command("networksetup", "-setwebproxystate", s, "off").Run()
		_ = exec.Command("networksetup", "-setsecurewebproxystate", s, "off").Run()
		_ = exec.Command("networksetup", "-setsocksfirewallproxystate", s, "off").Run()
	}
	return nil
}
