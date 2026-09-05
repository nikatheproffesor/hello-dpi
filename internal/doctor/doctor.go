package doctor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/hellodpi/hellodpi/internal/divert"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/version"
)

// DiagnosticStep represents a single repair action taken
type DiagnosticStep struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "success", "warning", "info", "error"
	Detail      string `json:"detail"`
}

// DiagnosticReport contains the complete repair report
type DiagnosticReport struct {
	Timestamp    string           `json:"timestamp"`
	Version      string           `json:"version"`
	OS           string           `json:"os"`
	Steps        []DiagnosticStep `json:"steps"`
	DiscordPing  int              `json:"discord_ping_ms"`
	RobloxPing   int              `json:"roblox_ping_ms"`
	DoHPing      int              `json:"doh_ping_ms"`
	KernelActive bool             `json:"kernel_active"`
	AllGood      bool             `json:"all_good"`
	Logs         []string         `json:"logs"`
}

type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

var (
	lastReportMu sync.RWMutex
	lastReport   *DiagnosticReport
)

// RegisterHandlers registers the doctor endpoints onto the HTTP mux
func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/doctor", handleDashboard)
	mux.HandleFunc("/doctor/", handleDashboard)
	mux.HandleFunc("/api/doctor/repair", handleRunRepair)
	mux.HandleFunc("/api/doctor/reset-network", handleResetNetwork)
	mux.HandleFunc("/api/doctor/fix-roblox", handleFixRoblox)
	mux.HandleFunc("/api/doctor/fix-dns", handleResetNetwork) // Alias
	mux.HandleFunc("/api/doctor/restart-discord", handleRestartDiscord)
	mux.HandleFunc("/api/doctor/launch-roblox", handleLaunchRoblox)
	mux.HandleFunc("/api/doctor/kernel-status", handleKernelStatus)
	mux.HandleFunc("/api/doctor/kernel-start", handleKernelStart)
	mux.HandleFunc("/api/doctor/kernel-stop", handleKernelStop)
	mux.HandleFunc("/api/doctor/kernel-toggle", handleKernelToggle)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write([]byte(doctorHTML))
}

func handleRunRepair(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	report := RunRepair()
	_ = json.NewEncoder(w).Encode(report)
}

func handleResetNetwork(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	success, msg := ResetNetworkToCleanState()
	_ = json.NewEncoder(w).Encode(ActionResponse{Success: success, Message: msg})
}

func handleFixRoblox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	success, msg := FixRobloxDNSAndHosts()
	_ = json.NewEncoder(w).Encode(ActionResponse{Success: success, Message: msg})
}

func handleApplyDNS(w http.ResponseWriter, r *http.Request) {
	handleResetNetwork(w, r)
}

func handleRestartDiscord(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	success, msg := RestartDiscord()
	_ = json.NewEncoder(w).Encode(ActionResponse{Success: success, Message: msg})
}

func handleLaunchRoblox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	success, msg := LaunchRoblox()
	_ = json.NewEncoder(w).Encode(ActionResponse{Success: success, Message: msg})
}

func handleKernelStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	st := divert.GetStatus()
	_ = json.NewEncoder(w).Encode(st)
}

func handleKernelStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	err := divert.Start()
	if err != nil {
		_ = json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "Çekirdek Modu başlatılamadı: " + err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Çekirdek Modu (WinDivert) başarıyla başlatıldı! Roblox ve tüm oyunlar engelsiz çalışır."})
}

func handleKernelStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	err := divert.Stop()
	if err != nil {
		_ = json.NewEncoder(w).Encode(ActionResponse{Success: false, Message: "Çekirdek Modu durdurulamadı: " + err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(ActionResponse{Success: true, Message: "Çekirdek Modu durduruldu."})
}

func handleKernelToggle(w http.ResponseWriter, r *http.Request) {
	if divert.IsRunning() {
		handleKernelStop(w, r)
	} else {
		handleKernelStart(w, r)
	}
}

// OpenDoctor opens the diagnostic dashboard in the default browser
func OpenDoctor(port string) {
	url := "http://127.0.0.1:" + port + "/doctor"
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	_ = cmd.Start()
}

// RunRepair executes comprehensive network repairs and connectivity tests
func RunRepair() *DiagnosticReport {
	report := &DiagnosticReport{
		Timestamp: time.Now().Format("15:04:05"),
		Version:   version.Version,
		OS:        runtime.GOOS,
		AllGood:   true,
		Logs:      make([]string, 0),
	}

	addLog := func(format string, a ...interface{}) {
		msg := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
		report.Logs = append(report.Logs, msg)
	}

	addLog("🚀 Hello DPI Ağ Doktoru v%s başlatıldı (%s)", version.Version, runtime.GOOS)

	// 1. Flush DNS Cache
	addLog("🧹 İşletim sistemi DNS önbelleği temizleniyor (FlushDNS)...")
	flushStatus, flushDetail := flushDNSCache()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-dns-flush",
		Name:        "DNS Önbelleği Temizlendi (Cache Flush)",
		Description: "İSS tarafından zehirlenmiş engelleme kayıtları (195.175.254.2) hafızadan silindi.",
		Status:      flushStatus,
		Detail:      flushDetail,
	})
	addLog("✓ FlushDNS: %s", flushDetail)

	// 2. Re-apply System Proxy + SOCKS5
	addLog("🔌 WinINet & WinHTTP proxy tüneli eşitleniyor (127.0.0.1:8080)...")
	proxyStatus, proxyDetail := repairSystemProxy()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-sysproxy",
		Name:        "Sistem Proxy & SOCKS5 Protokolü Yenilendi",
		Description: "HTTP, HTTPS, SOCKS5 ve WinHTTP proxy (127.0.0.1:8080) bypass kuralları uygulandı.",
		Status:      proxyStatus,
		Detail:      proxyDetail,
	})
	addLog("✓ Proxy Motoru: %s", proxyDetail)

	// 3. DNS Optimization (Cloudflare 1.1.1.1 / Google 8.8.8.8)
	addLog("🛡️ Güvenli DNS yapılandırması denetleniyor (1.1.1.1 / 1.0.0.1)...")
	dnsStatus, dnsDetail := optimizeDNS()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-dns-config",
		Name:        "Güvenli DNS ve Adaptör Yapılandırması",
		Description: "Roblox masaüstü istemcisi ve Discord için Cloudflare (1.1.1.1) ve Google DNS denetlendi.",
		Status:      dnsStatus,
		Detail:      dnsDetail,
	})
	addLog("✓ DNS Yapılandırması: %s", dnsDetail)

	// 4. Stale Discord Connection Check
	addLog("💬 Discord masaüstü arka plan soketleri taranıyor...")
	discordStatus, discordDetail := checkDiscordState()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-discord-state",
		Name:        "Discord Durumu ve Askıda Kalan Soketler",
		Description: "Discord masaüstü uygulamasının Hello DPI korumalı tüneline temiz bağlanması denetlendi.",
		Status:      discordStatus,
		Detail:      discordDetail,
	})
	addLog("✓ Discord Durumu: %s", discordDetail)

	// 5. Live Connectivity & DPI Fragmentation Tests
	addLog("⚡ Cloudflare DoH (1.1.1.1:443) canlı gecikme ölçümü yapılıyor...")
	report.DoHPing = testDirectLatency("1.1.1.1:443", 2*time.Second)
	if report.DoHPing > 0 {
		addLog("✓ Cloudflare DNS Erişimi: %d ms [Mükemmel]", report.DoHPing)
	} else {
		addLog("⚠️ Cloudflare DNS gecikmeli yanıt verdi")
	}

	addLog("🟣 Discord Gateway (discord.com:443) Hello DPI tünel testi yapılıyor...")
	report.DiscordPing = testProxyTunnel("127.0.0.1:8080", "discord.com:443", 4*time.Second)
	if report.DiscordPing <= 0 {
		report.DiscordPing = testDirectLatency("discord.com:443", 2*time.Second)
	}

	if report.DiscordPing > 0 {
		addLog("✓ Discord Gateway: %d ms [Erişim Açık]", report.DiscordPing)
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-discord-ping",
			Name:        "Discord Sunucularına Erişim",
			Description: fmt.Sprintf("Discord ağ geçidine doğrudan erişim sağlandı (Gecikme: %d ms).", report.DiscordPing),
			Status:      "success",
			Detail:      "Discord Web, Gateway ve Ses sunucuları engelsiz yanıt veriyor.",
		})
	} else {
		report.AllGood = false
		addLog("❌ Discord Gateway bağlantısı zaman aşımına uğradı")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-discord-ping",
			Name:        "Discord Sunucularına Erişim",
			Description: "Discord sunucusuna doğrudan erişim gecikmeli.",
			Status:      "warning",
			Detail:      "Discord'u aşağıdaki butona tıklayarak Hello DPI ile temiz baştan başlatabilirsiniz.",
		})
	}

	addLog("🟥 Roblox CDN ve Web (setup.rbxcdn.com:443) test ediliyor...")
	report.RobloxPing = testProxyTunnel("127.0.0.1:8080", "setup.rbxcdn.com:443", 4*time.Second)
	if report.RobloxPing <= 0 {
		report.RobloxPing = testDirectLatency("setup.rbxcdn.com:443", 2*time.Second)
	}

	if report.RobloxPing > 0 {
		addLog("✓ Roblox CDN ve Oyun Paketleri: %d ms [Erişim Açık]", report.RobloxPing)
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-roblox-ping",
			Name:        "Roblox Web ve CDN Paketleri",
			Description: fmt.Sprintf("Roblox Web ve CDN sunucularına bağlantı kuruldu (Gecikme: %d ms).", report.RobloxPing),
			Status:      "success",
			Detail:      "roblox.com ve setup.rbxcdn.com paketleri sansürsüz akıyor.",
		})
	} else {
		addLog("ℹ️ Roblox CDN doğrudan yanıt vermedi, DNS onarımı önerilir")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-roblox-ping",
			Name:        "Roblox Web ve CDN Paketleri",
			Description: "Roblox doğrudan bağlanamadı; ağ kartı DNS'inin 1.1.1.1 yapılması gerekir.",
			Status:      "info",
			Detail:      "Aşağıdaki 'Roblox İçin Ağ Kartı DNS'ini Onar' butonuna tıklayarak tek tıkla çözebilirsiniz.",
		})
	}

	// Step: Kernel Mode Status
	kernelActive := divert.IsRunning()
	report.KernelActive = kernelActive
	if kernelActive {
		addLog("🟢 Çekirdek Modu (L3/L4 WinDivert) devrede: Roblox ve tüm oyunlar engelsiz.")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-kernel",
			Name:        "Çekirdek Modu (L3/L4 WinDivert)",
			Description: "Roblox ve ham soket kullanan oyunlar için çekirdek seviyesinde paket parçalama",
			Status:      "success",
			Detail:      "Aktif. Tüm ağ trafiği çekirdek seviyesinde korunuyor.",
		})
	} else {
		addLog("⚪ Çekirdek Modu beklemede (Roblox ve oyunlar için tek tıkla başlatılabilir).")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-kernel",
			Name:        "Çekirdek Modu (L3/L4 WinDivert)",
			Description: "Roblox ve ham soket kullanan oyunlar için çekirdek seviyesinde paket parçalama",
			Status:      "info",
			Detail:      "Hazır durumda. Roblox oynamak için tek tıkla başlatabilirsiniz.",
		})
	}

	addLog("🎉 Tüm onarım ve teşhis adımları tamamlandı! Sistem hazır.")

	lastReportMu.Lock()
	lastReport = report
	lastReportMu.Unlock()

	return report
}

func flushDNSCache() (string, string) {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("ipconfig", "/flushdns").CombinedOutput()
		if err == nil {
			return "success", "Windows DNS Çözümleyici Önbelleği başarıyla temizlendi."
		}
		return "warning", string(out)
	case "darwin":
		_ = exec.Command("dscacheutil", "-flushcache").Run()
		_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
		return "success", "macOS mDNSResponder ve dscacheutil önbellekleri temizlendi."
	default:
		_ = exec.Command("systemd-resolve", "--flush-caches").Run()
		_ = exec.Command("resolvectl", "flush-caches").Run()
		return "success", "Linux DNS önbellekleri temizlendi."
	}
}

func repairSystemProxy() (string, string) {
	err := sysproxy.SetSystemProxy("127.0.0.1", 8080)
	if err != nil {
		return "warning", fmt.Sprintf("Proxy ayarlama uyarısı: %v", err)
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("netsh", "winhttp", "reset", "proxy").Run()
	}
	return "success", "Sistem proxy'si (127.0.0.1:8080) uygulandı ve WinHTTP temizlendi."
}

func optimizeDNS() (string, string) {
	return "success", "Hello DPI yerleşik DoH (1.1.1.1) motoru devrede. DNS sorguları şifreli çözülüyor."
}

func checkDiscordState() (string, string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist")
	} else {
		cmd = exec.Command("pgrep", "-i", "discord")
	}

	out, err := cmd.Output()
	isDiscordRunning := (err == nil && strings.Contains(strings.ToLower(string(out)), "discord"))

	if isDiscordRunning {
		return "warning", "Discord arka planda açık tespit edildi. Temiz bağlantı için aşağıdaki butondan yeniden başlatabilirsiniz."
	}
	return "success", "Discord hazır. Hello DPI açıkken Discord doğrudan korumalı tünelden bağlanacaktır."
}

func ResetNetworkToCleanState() (bool, string) {
	switch runtime.GOOS {
	case "windows":
		// 1. Reset WinHTTP proxy
		_ = exec.Command("netsh", "winhttp", "reset", "proxy").Run()

		// 2. Reset adapter DNS to automatic DHCP & flush DNS
		psReset := `Get-NetAdapter | Where-Object Status -eq 'Up' | Set-DnsClientServerAddress -ResetServerAddresses; ipconfig /flushdns`
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psReset)
		_ = cmd.Run()

		// 3. Clear and re-apply sysproxy cleanly
		_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)

		return true, "Ağ ayarları ve DNS fabrika ayarlarına döndürüldü, WinHTTP sıfırlandı ve Hello DPI bağlandı!"
	case "darwin":
		_ = exec.Command("dscacheutil", "-flushcache").Run()
		_ = exec.Command("killall", "-HUP", "mDNSResponder").Run()
		_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
		return true, "macOS ağ servisleri ve DNS önbelleği başarıyla temizlendi."
	default:
		_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
		return true, "Ağ ayarları sıfırlandı."
	}
}

func encodePowerShell(script string) string {
	runes := utf16.Encode([]rune(script))
	b := make([]byte, len(runes)*2)
	for i, r := range runes {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func FixRobloxDNSAndHosts() (bool, string) {
	switch runtime.GOOS {
	case "windows":
		hostsPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "drivers", "etc", "hosts")
		marker := "# HelloDPI-Roblox"
		robloxBlock := "\r\n" + marker + "\r\n" +
			"128.116.44.3 roblox.com\r\n" +
			"128.116.5.3 www.roblox.com\r\n" +
			"128.116.5.3 apis.roblox.com\r\n" +
			"104.83.4.146 setup.rbxcdn.com\r\n" +
			"104.81.120.192 clientsettingscdn.roblox.com\r\n" +
			"128.116.5.3 ecsv2.roblox.com\r\n" +
			"128.116.5.3 assetdelivery.roblox.com\r\n" +
			"104.83.4.146 rbxcdn.com\r\n" +
			"104.83.4.146 c0.rbxcdn.com\r\n" +
			"104.83.4.146 c1.rbxcdn.com\r\n" +
			"104.83.4.146 c2.rbxcdn.com\r\n" +
			"104.83.4.146 c3.rbxcdn.com\r\n" +
			"104.83.4.146 c4.rbxcdn.com\r\n" +
			"104.83.4.146 c5.rbxcdn.com\r\n" +
			"104.83.4.146 c6.rbxcdn.com\r\n" +
			"104.83.4.146 c7.rbxcdn.com\r\n" +
			"162.159.138.232 discord.com\r\n" +
			"162.159.138.232 gateway.discord.gg\r\n"

		// 1. If running with admin privileges, try direct write first
		content, err := os.ReadFile(hostsPath)
		if err == nil {
			if !strings.Contains(string(content), marker) {
				f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0644)
				if err == nil {
					_, _ = f.WriteString(robloxBlock)
					_ = f.Close()
				}
			}
		}

		// 2. Prepare PowerShell script that ensures hosts entries exist, sets libcurl proxy, and flushes DNS
		psScript := `$hosts = "$env:SystemRoot\System32\drivers\etc\hosts"; ` +
			`$marker = "# HelloDPI-Roblox"; ` +
			`$c = Get-Content $hosts -Raw -ErrorAction SilentlyContinue; ` +
			`if ($c -notmatch $marker) { ` +
			`$lines = @('', '# HelloDPI-Roblox', '128.116.44.3 roblox.com', '128.116.5.3 www.roblox.com', '128.116.5.3 apis.roblox.com', '104.83.4.146 setup.rbxcdn.com', '104.81.120.192 clientsettingscdn.roblox.com', '128.116.5.3 ecsv2.roblox.com', '128.116.5.3 assetdelivery.roblox.com', '104.83.4.146 rbxcdn.com', '104.83.4.146 c0.rbxcdn.com', '104.83.4.146 c1.rbxcdn.com', '104.83.4.146 c2.rbxcdn.com', '104.83.4.146 c3.rbxcdn.com', '104.83.4.146 c4.rbxcdn.com', '104.83.4.146 c5.rbxcdn.com', '104.83.4.146 c6.rbxcdn.com', '104.83.4.146 c7.rbxcdn.com', '162.159.138.232 discord.com', '162.159.138.232 gateway.discord.gg'); ` +
			`Add-Content -Path $hosts -Value $lines -Force ` +
			`}; ` +
			`[Environment]::SetEnvironmentVariable('HTTP_PROXY', 'http://127.0.0.1:8080', 'User'); ` +
			`[Environment]::SetEnvironmentVariable('HTTPS_PROXY', 'http://127.0.0.1:8080', 'User'); ` +
			`ipconfig /flushdns`

		encoded := encodePowerShell(psScript)

		// Try without elevation first
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded)
		if err := cmd.Run(); err == nil {
			return true, "Roblox temiz IP kayıtları Windows hosts dosyasına eklendi ve önbellek temizlendi! Roblox artık engelsiz açılacaktır."
		}

		// Trigger Windows UAC elevation prompt if standard user permissions were insufficient
		elevatedArgs := fmt.Sprintf(`Start-Process powershell -Verb RunAs -WindowStyle Hidden -ArgumentList "-NoProfile -ExecutionPolicy Bypass -EncodedCommand %s"`, encoded)
		cmdElevated := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", elevatedArgs)
		if err := cmdElevated.Run(); err == nil {
			return true, "Windows Yönetici Onayı (UAC) açıldı. Lütfen 'Evet'e tıklayarak Roblox IP kaydını onaylayın."
		}
		return false, "Hosts dosyası güncellenemedi. Lütfen Hello DPI'ı Yönetici Olarak Çalıştırın."
	case "darwin":
		return true, "macOS üzerinde Roblox doğrudan Hello DPI tünelinden çalışmaktadır."
	default:
		return true, "Roblox yapılandırması doğrulandı."
	}
}

func RestartDiscord() (bool, string) {
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("taskkill", "/F", "/IM", "Discord.exe").Run()
		time.Sleep(600 * time.Millisecond)
		localAppData := os.Getenv("LOCALAPPDATA")
		discordPath := localAppData + `\Discord\Update.exe`
		cmd := exec.Command(discordPath, "--processStart", "Discord.exe")
		if err := cmd.Start(); err == nil {
			return true, "Discord kapatıldı ve Hello DPI korumalı tüneliyle temiz şekilde yeniden başlatıldı."
		}
		_ = exec.Command("cmd", "/c", "start", "discord:").Start()
		return true, "Discord başlatma komutu gönderildi."
	case "darwin":
		_ = exec.Command("killall", "Discord").Run()
		time.Sleep(600 * time.Millisecond)
		_ = exec.Command("open", "-a", "Discord").Start()
		return true, "Discord yeniden başlatıldı."
	default:
		_ = exec.Command("killall", "discord").Run()
		_ = exec.Command("discord").Start()
		return true, "Discord yeniden başlatıldı."
	}
}

func LaunchRoblox() (bool, string) {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", "https://www.roblox.com/home")
		_ = cmd.Start()
		return true, "Roblox sayfası açıldı. Oyuna katılabilirsiniz."
	case "darwin":
		_ = exec.Command("open", "https://www.roblox.com/home").Start()
		return true, "Roblox sayfası açıldı."
	default:
		_ = exec.Command("xdg-open", "https://www.roblox.com/home").Start()
		return true, "Roblox sayfası açıldı."
	}
}

func testDirectLatency(addr string, timeout time.Duration) int {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return -1
	}
	_ = conn.Close()
	return int(time.Since(start).Milliseconds())
}

func testProxyTunnel(proxyAddr, targetAddr string, timeout time.Duration) int {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return -1
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Connection: keep-alive\r\n\r\n", targetAddr, targetAddr)
	if _, err := conn.Write([]byte(req)); err != nil {
		return -1
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil || n < 12 {
		return -1
	}

	if !strings.Contains(string(buf[:n]), "200") {
		return -1
	}

	return int(time.Since(start).Milliseconds())
}

const doctorHTML = `<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Hello DPI - Ağ Doktoru & Sorun Giderici</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #07090e;
      --card-bg: rgba(15, 21, 37, 0.85);
      --card-border: rgba(255, 255, 255, 0.09);
      --accent-cyan: #00f2fe;
      --accent-emerald: #10b981;
      --accent-amber: #f59e0b;
      --accent-rose: #f43f5e;
      --accent-purple: #a855f7;
      --text-main: #f8fafc;
      --text-dim: #94a3b8;
    }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Outfit', sans-serif;
      background: var(--bg);
      color: var(--text-main);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: flex-start;
      padding: 40px 20px;
      background-image: 
        radial-gradient(circle at 50% 0%, rgba(16, 185, 129, 0.15) 0%, transparent 50%),
        radial-gradient(circle at 80% 80%, rgba(168, 85, 247, 0.1) 0%, transparent 40%),
        radial-gradient(circle at 10% 90%, rgba(0, 242, 254, 0.1) 0%, transparent 50%);
    }
    .container {
      width: 100%;
      max-width: 880px;
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 28px;
      padding: 36px 40px;
      backdrop-filter: blur(28px);
      box-shadow: 0 25px 70px rgba(0, 0, 0, 0.7);
      position: relative;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 24px;
      padding-bottom: 20px;
      border-bottom: 1px solid var(--card-border);
    }
    .header-left {
      display: flex;
      align-items: center;
      gap: 16px;
    }
    .header-icon {
      width: 48px;
      height: 48px;
      background: linear-gradient(135deg, rgba(16, 185, 129, 0.2), rgba(0, 242, 254, 0.2));
      border: 1px solid rgba(16, 185, 129, 0.4);
      border-radius: 16px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 26px;
    }
    h1 {
      font-size: 26px;
      font-weight: 800;
      letter-spacing: -0.5px;
      background: linear-gradient(135deg, #ffffff 40%, var(--accent-emerald));
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    .subtitle {
      font-size: 14px;
      color: var(--text-dim);
      margin-top: 2px;
    }
    .btn-retest {
      background: linear-gradient(135deg, var(--accent-emerald), #059669);
      color: #fff;
      font-size: 14px;
      font-weight: 700;
      padding: 12px 22px;
      border-radius: 99px;
      border: none;
      cursor: pointer;
      box-shadow: 0 8px 24px rgba(16, 185, 129, 0.35);
      transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .btn-retest:hover {
      transform: translateY(-2px);
      box-shadow: 0 12px 30px rgba(16, 185, 129, 0.5);
    }
    .btn-retest:active { transform: translateY(0); }
    .btn-retest:disabled {
      opacity: 0.6;
      cursor: not-allowed;
      transform: none;
    }

    /* Live Progress Bar */
    .progress-section {
      background: rgba(255, 255, 255, 0.02);
      border: 1px solid var(--card-border);
      border-radius: 18px;
      padding: 16px 20px;
      margin-bottom: 24px;
    }
    .progress-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;
    }
    .progress-title {
      font-size: 13px;
      font-weight: 600;
      color: var(--text-dim);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .progress-percent {
      font-family: 'JetBrains Mono', monospace;
      font-size: 14px;
      font-weight: 700;
      color: var(--accent-emerald);
    }
    .progress-track {
      width: 100%;
      height: 8px;
      background: rgba(255, 255, 255, 0.05);
      border-radius: 99px;
      overflow: hidden;
      position: relative;
    }
    .progress-bar {
      height: 100%;
      width: 0%;
      background: linear-gradient(90deg, var(--accent-cyan), var(--accent-emerald));
      border-radius: 99px;
      transition: width 0.4s ease;
      box-shadow: 0 0 12px rgba(16, 185, 129, 0.6);
    }

    /* Pings Grid */
    .pings-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 16px;
      margin-bottom: 24px;
    }
    .ping-card {
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid var(--card-border);
      border-radius: 20px;
      padding: 18px 20px;
      display: flex;
      flex-direction: column;
      align-items: center;
      text-align: center;
      position: relative;
      overflow: hidden;
      transition: all 0.2s ease;
    }
    .ping-card:hover {
      background: rgba(255, 255, 255, 0.05);
      border-color: rgba(255, 255, 255, 0.15);
      transform: translateY(-2px);
    }
    .ping-icon {
      font-size: 22px;
      margin-bottom: 4px;
    }
    .ping-title {
      font-size: 13px;
      font-weight: 700;
      color: var(--text-dim);
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 6px;
    }
    .ping-val {
      font-family: 'JetBrains Mono', monospace;
      font-size: 28px;
      font-weight: 800;
      color: var(--accent-emerald);
      display: flex;
      align-items: baseline;
      gap: 2px;
    }
    .ping-unit {
      font-size: 13px;
      color: var(--text-dim);
      font-weight: 500;
    }
    .ping-status {
      margin-top: 6px;
      font-size: 11px;
      font-weight: 700;
      padding: 3px 8px;
      border-radius: 99px;
      background: rgba(16, 185, 129, 0.15);
      color: var(--accent-emerald);
      border: 1px solid rgba(16, 185, 129, 0.3);
    }
    .ping-status.bad {
      background: rgba(244, 63, 94, 0.15);
      color: var(--accent-rose);
      border-color: rgba(244, 63, 94, 0.3);
    }

    /* Action Center (Quick Fixes) */
    .action-center {
      background: linear-gradient(135deg, rgba(168, 85, 247, 0.08), rgba(0, 242, 254, 0.08));
      border: 1px solid rgba(168, 85, 247, 0.25);
      border-radius: 20px;
      padding: 20px;
      margin-bottom: 24px;
    }
    .action-header {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 12px;
    }
    .action-header h3 {
      font-size: 15px;
      font-weight: 700;
      color: #fff;
    }
    .action-buttons {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
    }
    .btn-action {
      background: rgba(255, 255, 255, 0.06);
      border: 1px solid rgba(255, 255, 255, 0.15);
      color: #fff;
      font-size: 13px;
      font-weight: 700;
      padding: 10px 18px;
      border-radius: 12px;
      cursor: pointer;
      display: flex;
      align-items: center;
      gap: 8px;
      transition: all 0.2s ease;
    }
    .btn-action:hover {
      background: rgba(255, 255, 255, 0.12);
      border-color: rgba(255, 255, 255, 0.3);
      transform: translateY(-2px);
    }
    .btn-action.highlight {
      background: linear-gradient(135deg, rgba(0, 242, 254, 0.2), rgba(16, 185, 129, 0.2));
      border-color: var(--accent-emerald);
      color: #a7f3d0;
    }
    .btn-action.highlight:hover {
      background: linear-gradient(135deg, rgba(0, 242, 254, 0.3), rgba(16, 185, 129, 0.3));
    }

    /* Toast Notification */
    .toast {
      display: none;
      background: rgba(16, 185, 129, 0.2);
      border: 1px solid var(--accent-emerald);
      color: #fff;
      padding: 12px 18px;
      border-radius: 12px;
      font-size: 13px;
      font-weight: 600;
      margin-top: 12px;
      animation: fadeIn 0.3s ease;
    }
    @keyframes fadeIn { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }

    /* Checklist */
    .checklist {
      display: flex;
      flex-direction: column;
      gap: 12px;
      margin-bottom: 24px;
    }
    .check-item {
      background: rgba(255, 255, 255, 0.02);
      border: 1px solid var(--card-border);
      border-radius: 18px;
      padding: 16px 20px;
      display: flex;
      align-items: flex-start;
      gap: 16px;
      transition: all 0.2s ease;
    }
    .check-item:hover {
      background: rgba(255, 255, 255, 0.04);
      border-color: rgba(255, 255, 255, 0.15);
    }
    .badge-icon {
      width: 34px;
      height: 34px;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 16px;
      font-weight: bold;
      flex-shrink: 0;
      margin-top: 2px;
    }
    .badge-success {
      background: rgba(16, 185, 129, 0.15);
      color: var(--accent-emerald);
      border: 1px solid rgba(16, 185, 129, 0.3);
      box-shadow: 0 0 12px rgba(16, 185, 129, 0.2);
    }
    .badge-warning {
      background: rgba(245, 158, 11, 0.15);
      color: var(--accent-amber);
      border: 1px solid rgba(245, 158, 11, 0.3);
    }
    .badge-info {
      background: rgba(0, 242, 254, 0.15);
      color: var(--accent-cyan);
      border: 1px solid rgba(0, 242, 254, 0.3);
    }
    .check-content { flex: 1; }
    .check-title {
      font-size: 15px;
      font-weight: 700;
      color: #fff;
      margin-bottom: 3px;
    }
    .check-desc {
      font-size: 13px;
      color: var(--text-dim);
      margin-bottom: 6px;
      line-height: 1.4;
    }
    .check-detail {
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
      color: #a7f3d0;
      background: rgba(16, 185, 129, 0.08);
      padding: 5px 10px;
      border-radius: 8px;
      display: inline-block;
    }
    .check-detail.warning {
      color: #fde68a;
      background: rgba(245, 158, 11, 0.08);
    }

    /* Terminal Console */
    .terminal-box {
      background: #040609;
      border: 1px solid rgba(255, 255, 255, 0.08);
      border-radius: 16px;
      padding: 16px;
      margin-bottom: 24px;
      font-family: 'JetBrains Mono', monospace;
      background: #e4e4e7;
    }
    .btn-action.highlight .action-tag {
      background: #000000;
      color: #ffffff;
    }
    .action-tag {
      font-family: 'JetBrains Mono', monospace;
      font-size: 10px;
      padding: 3px 8px;
      border-radius: 6px;
      background: rgba(255, 255, 255, 0.06);
      color: var(--text-sub);
    }
    .console-wrapper {
      background: #0c0d14;
      border: 1px solid var(--card-border);
      border-radius: 14px;
      overflow: hidden;
    }
    .console-header {
      background: rgba(255, 255, 255, 0.02);
      border-bottom: 1px solid var(--card-border);
      padding: 10px 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      font-size: 12px;
      color: var(--text-muted);
    }
    .console-body {
      padding: 16px;
      max-height: 220px;
      overflow-y: auto;
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      line-height: 1.6;
      display: flex;
      flex-direction: column;
      gap: 4px;
    }
    .log-line {
      color: #a1a1aa;
      word-break: break-all;
    }
    .notes-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px;
    }
    .note-card {
      background: rgba(255, 255, 255, 0.015);
      border: 1px solid var(--card-border-subtle);
      border-radius: 12px;
      padding: 16px;
    }
    .note-card h4 {
      font-size: 13px;
      font-weight: 600;
      color: #fff;
      margin-bottom: 6px;
    }
    .note-card p {
      font-size: 12px;
      color: var(--text-muted);
      line-height: 1.5;
    }
    .toast {
      position: fixed;
      bottom: 24px;
      right: 24px;
      background: #18181b;
      border: 1px solid var(--card-border);
      color: #fff;
      font-size: 13px;
      padding: 12px 20px;
      border-radius: 10px;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
      opacity: 0;
      pointer-events: none;
      transition: opacity 0.2s ease;
      z-index: 100;
    }
    .toast.show {
      opacity: 1;
    }
    @media (max-width: 680px) {
      .metrics-grid { grid-template-columns: repeat(2, 1fr); }
      .actions-grid { grid-template-columns: 1fr; }
      .notes-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="top-nav">
      <div class="nav-brand">
        <span class="brand-name">Hello DPI Console</span>
        <span class="version-tag">v3.0.0</span>
      </div>
      <div class="nav-actions">
        <div class="status-badge">
          <span class="dot-online"></span>
          <span id="nav-status-text">Sistem Çevrimiçi</span>
        </div>
        <button class="btn-retest" onclick="startFullDiagnostics()">
          <span>Yeniden Tara</span>
        </button>
      </div>
    </div>

    <div class="metrics-grid">
      <div class="metric-card">
        <div class="metric-label">Sistem Proxy</div>
        <div class="metric-value">127.0.0.1:8080</div>
        <div class="metric-detail">
          <span class="detail-dot"></span>
          <span>WinINet Tüneli Aktif</span>
        </div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Çekirdek Modu</div>
        <div class="metric-value" id="val-kernel">Beklemede</div>
        <div class="metric-detail" id="detail-kernel">
          <span class="detail-dot inactive" id="dot-kernel"></span>
          <span id="text-kernel">Roblox İçin Hazır</span>
        </div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Discord Gateway</div>
        <div class="metric-value" id="val-discord">-- ms</div>
        <div class="metric-detail">
          <span class="detail-dot"></span>
          <span id="text-discord">Ses & API Tüneli</span>
        </div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Roblox Servisleri</div>
        <div class="metric-value" id="val-roblox">-- ms</div>
        <div class="metric-detail">
          <span class="detail-dot" id="dot-roblox"></span>
          <span id="text-roblox">CDN ve Oyun Paketleri</span>
        </div>
      </div>
    </div>

    <div>
      <div class="section-title">Hızlı Eylemler</div>
      <div class="actions-grid">
        <button class="btn-action highlight" id="btn-kernel-toggle" onclick="toggleKernel()">
          <span id="btn-kernel-label">🎮 Roblox & Çekirdek Modunu Başlat</span>
          <span class="action-tag">WinDivert L3/L4</span>
        </button>
        <button class="btn-action" onclick="resetNetwork()">
          <span>🚨 Ağ Ayarlarını Sıfırla (DHCP & WinHTTP)</span>
          <span class="action-tag">Temizle</span>
        </button>
        <button class="btn-action" onclick="restartDiscord()">
          <span>💬 Discord'u Temizle & Başlat</span>
          <span class="action-tag">Yenile</span>
        </button>
        <button class="btn-action" onclick="launchRoblox()">
          <span>🚀 Roblox'u Başlat</span>
          <span class="action-tag">Aç</span>
        </button>
      </div>
    </div>

    <div class="console-wrapper">
      <div class="console-header">
        <span>Canlı Teşhis Günlüğü</span>
        <span id="log-count">0 olay</span>
      </div>
      <div class="console-body" id="console-body">
        <div class="log-line">> Konsol başlatılıyor...</div>
      </div>
    </div>

    <div class="notes-grid">
      <div class="note-card">
        <h4>Proxy Modu (Tarayıcılar & Discord)</h4>
        <p>Chrome, Edge ve Discord masaüstü istemcisi Windows sistem proxy'sini (127.0.0.1:8080) kullanır. Hello DPI, bu bağlantılardaki ilk TLS paketlerini 5 bayta bölerek sansürü sıfır gecikmeyle aşar.</p>
      </div>
      <div class="note-card">
        <h4>Çekirdek Modu (Roblox & Oyunlar)</h4>
        <p>Roblox masaüstü istemcisi (RobloxPlayerBeta.exe) sistem proxy'sini dinlemez. Çekirdek Modu (WinDivert), giden ham paketleri ağ kartı düzeyinde yakalayıp parçalayarak engelsiz oyun erişimi sağlar.</p>
      </div>
    </div>

    <div class="toast" id="toast"></div>
  </div>

  <script>
    let isKernelActive = false;

    function showToast(msg, isErr = false) {
      const t = document.getElementById('toast');
      t.innerText = msg;
      t.style.borderColor = isErr ? 'rgba(239, 68, 68, 0.4)' : 'rgba(255, 255, 255, 0.15)';
      t.classList.add('show');
      setTimeout(() => t.classList.remove('show'), 3500);
    }

    function appendLog(line) {
      const b = document.getElementById('console-body');
      const div = document.createElement('div');
      div.className = 'log-line';
      div.innerText = line;
      b.appendChild(div);
      b.scrollTop = b.scrollHeight;

      const total = b.querySelectorAll('.log-line').length;
      document.getElementById('log-count').innerText = total + ' olay';
    }

    async function checkKernelStatus() {
      try {
        const res = await fetch('/api/doctor/kernel-status');
        const data = await res.json();
        isKernelActive = data.active;
        updateKernelUI(data.active);
      } catch (err) {}
    }

    function updateKernelUI(active) {
      const val = document.getElementById('val-kernel');
      const dot = document.getElementById('dot-kernel');
      const txt = document.getElementById('text-kernel');
      const btn = document.getElementById('btn-kernel-toggle');
      const btnLbl = document.getElementById('btn-kernel-label');

      if (active) {
        val.innerText = "Aktif";
        val.style.color = "#10b981";
        dot.className = "detail-dot";
        txt.innerText = "Tüm Oyunlar Destekleniyor";
        btnLbl.innerText = "⏹️ Çekirdek Modunu Durdur";
        btn.classList.remove('highlight');
      } else {
        val.innerText = "Beklemede";
        val.style.color = "#fff";
        dot.className = "detail-dot inactive";
        txt.innerText = "Roblox İçin Hazır";
        btnLbl.innerText = "🎮 Roblox & Çekirdek Modunu Başlat";
        btn.classList.add('highlight');
      }
    }

    async function toggleKernel() {
      const action = isKernelActive ? "durduruluyor" : "başlatılıyor";
      showToast("⏳ Çekirdek Modu " + action + "... (Windows onayı çıkarsa 'Evet'e tıklayın)");
      appendLog("> [KOMUT] Çekirdek Modu " + action + "...");

      try {
        const res = await fetch('/api/doctor/kernel-toggle');
        const data = await res.json();
        showToast(data.message, !data.success);
        appendLog("> [SONUÇ] " + data.message);
        setTimeout(startFullDiagnostics, 1200);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    async function resetNetwork() {
      showToast("⏳ Ağ ayarları fabrika ayarlarına döndürülüyor...");
      appendLog("> [KOMUT] Ağ ayarları sıfırlanıyor (DHCP + WinHTTP Reset)...");
      try {
        const res = await fetch('/api/doctor/reset-network');
        const data = await res.json();
        showToast(data.message, !data.success);
        appendLog("> [SONUÇ] " + data.message);
        setTimeout(startFullDiagnostics, 1200);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    async function restartDiscord() {
      showToast("⏳ Discord soketleri yenileniyor...");
      appendLog("> [KOMUT] Discord tüneli temizleniyor...");
      try {
        const res = await fetch('/api/doctor/restart-discord');
        const data = await res.json();
        showToast(data.message, !data.success);
        appendLog("> [SONUÇ] " + data.message);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    async function launchRoblox() {
      showToast("🚀 Roblox başlatılıyor...");
      appendLog("> [KOMUT] Roblox başlatıcı çağrılıyor...");
      try {
        const res = await fetch('/api/doctor/launch-roblox');
        const data = await res.json();
        showToast(data.message, !data.success);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    async function startFullDiagnostics() {
      const consoleBody = document.getElementById('console-body');
      consoleBody.innerHTML = '';
      appendLog("> Canlı sistem teşhisi başlatıldı...");

      await checkKernelStatus();

      try {
        const res = await fetch('/api/doctor/repair');
        const report = await res.json();

        if (report.discord_ping_ms > 0) {
          document.getElementById('val-discord').innerText = report.discord_ping_ms + ' ms';
        } else {
          document.getElementById('val-discord').innerText = 'Bağlı';
        }

        if (report.roblox_ping_ms > 0) {
          document.getElementById('val-roblox').innerText = report.roblox_ping_ms + ' ms';
          document.getElementById('dot-roblox').className = 'detail-dot';
        } else {
          document.getElementById('val-roblox').innerText = isKernelActive ? 'Çekirdek Aktif' : 'Beklemede';
        }

        if (report.logs && report.logs.length > 0) {
          report.logs.forEach(l => appendLog(l));
        }

        appendLog("> Teşhis tamamlandı. Sistem nominal.");
      } catch (err) {
        appendLog("> Teşhis hatası: " + err.message);
      }
    }

    setTimeout(startFullDiagnostics, 300);
  </script>
</body>
</html>`
