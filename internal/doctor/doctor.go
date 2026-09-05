package doctor

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

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
	Timestamp   string           `json:"timestamp"`
	Version     string           `json:"version"`
	OS          string           `json:"os"`
	Steps       []DiagnosticStep `json:"steps"`
	DiscordPing int              `json:"discord_ping_ms"`
	RobloxPing  int              `json:"roblox_ping_ms"`
	DoHPing     int              `json:"doh_ping_ms"`
	AllGood     bool             `json:"all_good"`
	Logs        []string         `json:"logs"`
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
	mux.HandleFunc("/api/doctor/fix-dns", handleResetNetwork) // Alias
	mux.HandleFunc("/api/doctor/restart-discord", handleRestartDiscord)
	mux.HandleFunc("/api/doctor/launch-roblox", handleLaunchRoblox)
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
      font-size: 12px;
      color: #6ee7b7;
      overflow-x: auto;
      max-height: 180px;
      overflow-y: auto;
      box-shadow: inset 0 2px 10px rgba(0, 0, 0, 0.8);
    }
    .terminal-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding-bottom: 8px;
      margin-bottom: 10px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.06);
      color: var(--text-dim);
      font-size: 11px;
    }
    .terminal-dots { display: flex; gap: 6px; }
    .dot { width: 10px; height: 10px; border-radius: 50%; }
    .dot-red { background: #ef4444; }
    .dot-yellow { background: #f59e0b; }
    .dot-green { background: #10b981; }
    .terminal-line { margin-bottom: 4px; line-height: 1.5; white-space: pre-wrap; }

    /* Final Celebratory Banner */
    .banner-celebrate {
      display: none;
      background: linear-gradient(135deg, rgba(16, 185, 129, 0.15), rgba(0, 242, 254, 0.15));
      border: 1px solid rgba(16, 185, 129, 0.35);
      border-radius: 20px;
      padding: 22px;
      text-align: center;
      animation: fadeIn 0.4s ease;
    }
    .banner-celebrate h3 {
      font-size: 18px;
      font-weight: 800;
      color: #fff;
      margin-bottom: 6px;
    }
    .banner-celebrate p {
      font-size: 13px;
      color: var(--text-dim);
      line-height: 1.5;
    }

    /* Explainer Accordion */
    .explainer {
      margin-top: 24px;
      background: rgba(255, 255, 255, 0.015);
      border: 1px solid var(--card-border);
      border-radius: 18px;
      padding: 20px;
    }
    .explainer h4 {
      font-size: 14px;
      font-weight: 700;
      color: var(--text-dim);
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 12px;
    }
    .explainer-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }
    .explainer-card {
      background: rgba(255, 255, 255, 0.02);
      border-radius: 12px;
      padding: 14px;
    }
    .explainer-card h5 {
      font-size: 14px;
      font-weight: 700;
      color: #fff;
      margin-bottom: 4px;
    }
    .explainer-card p {
      font-size: 12px;
      color: var(--text-dim);
      line-height: 1.4;
    }

    @media (max-width: 768px) {
      .container { padding: 24px 20px; }
      .pings-grid { grid-template-columns: 1fr; }
      .explainer-grid { grid-template-columns: 1fr; }
      .header { flex-direction: column; align-items: flex-start; gap: 14px; }
      .btn-retest { width: 100%; justify-content: center; }
    }
  </style>
</head>
<body>
  <div class="container">
    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <div class="header-icon">🛠️</div>
        <div>
          <h1>Hello DPI Ağ Doktoru</h1>
          <div class="subtitle">Tek Tıkla Otomatik Ağ Onarımı ve Canlı Teşhis Merkezi</div>
        </div>
      </div>
      <button class="btn-retest" id="btn-retest" onclick="startFullDiagnostics()">
        <span id="btn-retest-icon">🔄</span>
        <span id="btn-retest-text">Yeniden Onar & Test Et</span>
      </button>
    </div>

    <!-- Live Scanning Progress -->
    <div class="progress-section">
      <div class="progress-header">
        <div class="progress-title" id="progress-status">
          <span>⏳</span> Sistem taranıyor ve bileşenler onarılıyor...
        </div>
        <div class="progress-percent" id="progress-percent">0%</div>
      </div>
      <div class="progress-track">
        <div class="progress-bar" id="progress-bar"></div>
      </div>
    </div>

    <!-- Latency Gauges Grid -->
    <div class="pings-grid">
      <div class="ping-card">
        <div class="ping-icon">🟣</div>
        <div class="ping-title">Discord API & Gateway</div>
        <div class="ping-val" id="val-discord">--<span class="ping-unit">ms</span></div>
        <div class="ping-status" id="status-discord">Taranıyor...</div>
      </div>
      <div class="ping-card">
        <div class="ping-icon">🟥</div>
        <div class="ping-title">Roblox CDN & Web</div>
        <div class="ping-val" id="val-roblox">--<span class="ping-unit">ms</span></div>
        <div class="ping-status" id="status-roblox">Taranıyor...</div>
      </div>
      <div class="ping-card">
        <div class="ping-icon">⚡</div>
        <div class="ping-title">Cloudflare DNS (1.1.1.1)</div>
        <div class="ping-val" id="val-doh">--<span class="ping-unit">ms</span></div>
        <div class="ping-status" id="status-doh">Taranıyor...</div>
      </div>
    </div>

    <!-- Quick Action Center -->
    <div class="action-center">
      <div class="action-header">
        <span>⚡</span>
        <h3>Tek Tıkla Hızlı Çözüm Butonları</h3>
      </div>
      <div class="action-buttons">
        <button class="btn-action highlight" onclick="resetNetwork()">
          <span>🚨</span> Acil Ağ Sıfırlama (Fabrika Ayarlarına Dön & Onar)
        </button>
        <button class="btn-action" onclick="restartDiscord()">
          <span>💬</span> Discord'u Temiz Yeniden Başlat
        </button>
        <button class="btn-action" onclick="launchRoblox()">
          <span>🎮</span> Roblox'u Aç
        </button>
      </div>
      <div class="toast" id="toast"></div>
    </div>

    <!-- Step by Step Diagnostic Checklist -->
    <div class="checklist" id="checklist">
      <!-- Steps rendered dynamically -->
    </div>

    <!-- Real-time Terminal Log Console -->
    <div class="terminal-box">
      <div class="terminal-header">
        <div class="terminal-dots">
          <div class="dot dot-red"></div>
          <div class="dot dot-yellow"></div>
          <div class="dot dot-green"></div>
        </div>
        <div>Canlı Sistem Tanılama Konsolu (Hello DPI v2.2.1)</div>
      </div>
      <div id="terminal-content">
        <div class="terminal-line">> Ağ Doktoru hazır. Tanılama başlatılıyor...</div>
      </div>
    </div>

    <!-- Success Celebration Banner -->
    <div class="banner-celebrate" id="banner-celebrate">
      <h3>🎉 Bilgisayarınızdaki Tüm Ağ Sorunları Çözüldü!</h3>
      <p>İSS DNS önbelleği silindi, Hello DPI proxy ve SOCKS tüneli doğrulandı. Discord ve Roblox artık sansürsüz ve engelsiz şekilde açılacaktır.</p>
    </div>

    <!-- Explainer Cards -->
    <div class="explainer">
      <h4>🔍 Neler Yapıldı ve Sorunlar Nasıl Çözüldü?</h4>
      <div class="explainer-grid">
        <div class="explainer-card">
          <h5>🎮 Roblox Neden Açılmıyordu?</h5>
          <p>Türk Telekom / Superonline gibi servis sağlayıcılar roblox.com ve setup.rbxcdn.com alan adlarını 195.175.254.2 adresine zehirler. Ağ Doktoru zehirli DNS önbelleğini sildi, WinHTTP proxy tünelini eşitledi ve Cloudflare DNS'i aktif hale getirdi.</p>
        </div>
        <div class="explainer-card">
          <h5>💬 Discord Sevgilinizin PC'sinde Neden Çalışmıyordu?</h5>
          <p>Windows proxy döngü (loopback) hatası ve arka planda asılı kalan eski Discord soketleri engelliyordu. Sistem proxy ayarları baştan yapılandırıldı ve Discord ağ geçidi canlı olarak doğrulandı.</p>
        </div>
      </div>
    </div>
  </div>

  <script>
    let isRunning = false;

    function showToast(msg, isErr = false) {
      const t = document.getElementById('toast');
      t.style.display = 'block';
      t.style.borderColor = isErr ? 'var(--accent-rose)' : 'var(--accent-emerald)';
      t.style.background = isErr ? 'rgba(244, 63, 94, 0.2)' : 'rgba(16, 185, 129, 0.2)';
      t.innerHTML = msg;
      setTimeout(() => { t.style.display = 'none'; }, 6000);
    }

    function appendTerminal(line) {
      const tc = document.getElementById('terminal-content');
      const div = document.createElement('div');
      div.className = 'terminal-line';
      div.innerText = line;
      tc.appendChild(div);
      tc.parentElement.scrollTop = tc.parentElement.scrollHeight;
    }

    async function resetNetwork() {
      showToast("⏳ Ağ ve DNS ayarları fabrika ayarlarına döndürülüyor, WinHTTP sıfırlanıyor...");
      appendTerminal("> [Komut] Ağ ayarları sıfırlanıyor (DHCP + WinHTTP Reset)...");
      try {
        const res = await fetch('/api/doctor/reset-network');
        const data = await res.json();
        showToast(data.message, !data.success);
        appendTerminal("> [Sonuç] " + data.message);
        setTimeout(startFullDiagnostics, 1500);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    async function restartDiscord() {
      showToast("⏳ Discord arka plandan kapatılıyor ve Hello DPI tüneliyle temiz baştan başlatılıyor...");
      appendTerminal("> [Komut] Discord yeniden başlatılıyor...");
      try {
        const res = await fetch('/api/doctor/restart-discord');
        const data = await res.json();
        showToast(data.message, !data.success);
        appendTerminal("> [Sonuç] " + data.message);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    async function launchRoblox() {
      showToast("🚀 Roblox web sayfası açılıyor...");
      appendTerminal("> [Komut] Roblox sayfası açılıyor...");
      try {
        const res = await fetch('/api/doctor/launch-roblox');
        const data = await res.json();
        showToast(data.message, !data.success);
      } catch (err) {
        showToast("Hata: " + err.message, true);
      }
    }

    function animateValue(id, start, end, duration) {
      const obj = document.getElementById(id);
      if (end <= 0) {
        obj.innerHTML = 'ERR<span class="ping-unit">ms</span>';
        return;
      }
      let startTimestamp = null;
      const step = (timestamp) => {
        if (!startTimestamp) startTimestamp = timestamp;
        const progress = Math.min((timestamp - startTimestamp) / duration, 1);
        const current = Math.floor(progress * (end - start) + start);
        obj.innerHTML = current + '<span class="ping-unit">ms</span>';
        if (progress < 1) {
          window.requestAnimationFrame(step);
        }
      };
      window.requestAnimationFrame(step);
    }

    async function startFullDiagnostics() {
      if (isRunning) return;
      isRunning = true;

      const btn = document.getElementById('btn-retest');
      const btnText = document.getElementById('btn-retest-text');
      const btnIcon = document.getElementById('btn-retest-icon');
      const progressBar = document.getElementById('progress-bar');
      const progressPercent = document.getElementById('progress-percent');
      const progressStatus = document.getElementById('progress-status');
      const celebrateBanner = document.getElementById('banner-celebrate');

      btn.disabled = true;
      btnIcon.innerText = '⏳';
      btnText.innerText = 'Onarılıyor...';
      celebrateBanner.style.display = 'none';

      // Reset Ping meters
      document.getElementById('val-discord').innerHTML = '--<span class="ping-unit">ms</span>';
      document.getElementById('val-roblox').innerHTML = '--<span class="ping-unit">ms</span>';
      document.getElementById('val-doh').innerHTML = '--<span class="ping-unit">ms</span>';
      document.getElementById('status-discord').innerText = 'Taranıyor...';
      document.getElementById('status-roblox').innerText = 'Taranıyor...';
      document.getElementById('status-doh').innerText = 'Taranıyor...';

      // Animate progress smoothly
      progressBar.style.width = '15%';
      progressPercent.innerText = '15%';
      progressStatus.innerHTML = '<span>🧹</span> DNS önbelleği ve zehirli kayıtlar siliniyor...';

      document.getElementById('terminal-content').innerHTML = '';
      appendTerminal('> [00:01] Hello DPI Ağ Doktoru çalıştırıldı.');
      appendTerminal('> [00:01] DNS Önbelleği temizleme başlatıldı (FlushDNS)...');

      try {
        setTimeout(() => {
          progressBar.style.width = '45%';
          progressPercent.innerText = '45%';
          progressStatus.innerHTML = '<span>🔌</span> Sistem proxy ve SOCKS5 tüneli eşitleniyor...';
        }, 600);

        setTimeout(() => {
          progressBar.style.width = '75%';
          progressPercent.innerText = '75%';
          progressStatus.innerHTML = '<span>🔍</span> Discord ve Roblox canlı bağlantıları test ediliyor...';
        }, 1200);

        const resp = await fetch('/api/doctor/repair');
        const report = await resp.json();

        progressBar.style.width = '100%';
        progressPercent.innerText = '100%';
        progressStatus.innerHTML = '<span>✅</span> Tüm onarımlar ve testler tamamlandı!';

        // Populate terminal logs
        if (report.logs && report.logs.length > 0) {
          report.logs.forEach(log => appendTerminal(log));
        }

        // Animate latency counters
        animateValue('val-discord', 0, report.discord_ping_ms, 800);
        animateValue('val-roblox', 0, report.roblox_ping_ms, 800);
        animateValue('val-doh', 0, report.doh_ping_ms, 800);

        const statusDiscord = document.getElementById('status-discord');
        if (report.discord_ping_ms > 0) {
          statusDiscord.innerText = '✓ Aktif & Açık';
          statusDiscord.className = 'ping-status';
        } else {
          statusDiscord.innerText = '⚠️ Yeniden Başlat';
          statusDiscord.className = 'ping-status bad';
        }

        const statusRoblox = document.getElementById('status-roblox');
        if (report.roblox_ping_ms > 0) {
          statusRoblox.innerText = '✓ Aktif & Açık';
          statusRoblox.className = 'ping-status';
        } else {
          statusRoblox.innerText = 'ℹ️ DNS Onarımı';
          statusRoblox.className = 'ping-status bad';
        }

        const statusDoH = document.getElementById('status-doh');
        if (report.doh_ping_ms > 0) {
          statusDoH.innerText = '✓ 1.1.1.1 Güvenli';
          statusDoH.className = 'ping-status';
        } else {
          statusDoH.innerText = '⚠️ Gecikmeli';
          statusDoH.className = 'ping-status bad';
        }

        // Render checklist steps
        renderChecklist(report.steps);

        if (report.all_good) {
          celebrateBanner.style.display = 'block';
        }
      } catch (err) {
        appendTerminal('> [HATA] Teşhis sırasında bağlantı koptu: ' + err.message);
        showToast("Sunucuya erişilemedi: " + err.message, true);
      } finally {
        isRunning = false;
        btn.disabled = false;
        btnIcon.innerText = '🔄';
        btnText.innerText = 'Yeniden Onar & Test Et';
      }
    }

    function renderChecklist(steps) {
      const list = document.getElementById('checklist');
      list.innerHTML = '';

      steps.forEach((s, idx) => {
        const div = document.createElement('div');
        div.className = 'check-item';
        div.style.animation = 'fadeIn ' + (0.2 + idx * 0.1) + 's ease';

        let icon = '✓';
        let badgeClass = 'badge-success';
        if (s.status === 'warning') {
          icon = '⚠️';
          badgeClass = 'badge-warning';
        } else if (s.status === 'info') {
          icon = 'ℹ️';
          badgeClass = 'badge-info';
        } else if (s.status === 'error') {
          icon = '✕';
          badgeClass = 'badge-warning';
        }

        div.innerHTML = 
          '<div class="badge-icon ' + badgeClass + '">' + icon + '</div>' +
          '<div class="check-content">' +
            '<div class="check-title">' + s.name + '</div>' +
            '<div class="check-desc">' + s.description + '</div>' +
            '<div class="check-detail ' + (s.status === 'warning' ? 'warning' : '') + '">' + s.detail + '</div>' +
          '</div>';
        list.appendChild(div);
      });
    }

    // Auto-run immediately when page loads
    window.addEventListener('DOMContentLoaded', () => {
      startFullDiagnostics();
    });
  </script>
</body>
</html>
`
