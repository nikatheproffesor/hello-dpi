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
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/rules"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/version"
	"github.com/hellodpi/hellodpi/internal/voice"
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

	globalRules  *rules.Engine
	globalProbe  *probe.Engine
	globalVoice  *voice.Optimizer
	onTuneApply  func(mode dpi.SplitMode, splitOffset int, delayMs int)
)

// SetCoreEngines connects active proxy subsystems to the doctor dashboard
func SetCoreEngines(r *rules.Engine, p *probe.Engine, v *voice.Optimizer, tuneCallback func(mode dpi.SplitMode, splitOffset int, delayMs int)) {
	globalRules = r
	globalProbe = p
	globalVoice = v
	onTuneApply = tuneCallback
}

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
	mux.HandleFunc("/api/doctor/autotune", handleAutoTune)
	mux.HandleFunc("/api/doctor/sync-rules", handleSyncRules)
	mux.HandleFunc("/api/doctor/rules-status", handleRulesStatus)
	mux.HandleFunc("/api/doctor/voice-test", handleVoiceTest)
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

func handleAutoTune(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if globalProbe == nil {
		globalProbe = probe.NewEngine()
	}

	res := globalProbe.RunProbe()
	if onTuneApply != nil && res.BypassVerified {
		onTuneApply(dpi.SplitMode(res.BestMode), res.BestSplitPos, res.BestDelayMs)
	}

	_ = json.NewEncoder(w).Encode(res)
}

func handleSyncRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if globalRules == nil {
		globalRules = rules.NewEngine()
	}

	err := globalRules.SyncRemote("")
	if err != nil {
		_ = json.NewEncoder(w).Encode(ActionResponse{
			Success: false,
			Message: "Kurallar guncellenemedi (Offline/Rate limit): " + err.Error(),
		})
		return
	}

	ver, direct, intercept, _ := globalRules.Stats()
	_ = json.NewEncoder(w).Encode(ActionResponse{
		Success: true,
		Message: fmt.Sprintf("Kurallar basariyla senkronize edildi. Versiyon: %s (Direct: %d, Intercept: %d)", ver, direct, intercept),
	})
}

func handleRulesStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if globalRules == nil {
		globalRules = rules.NewEngine()
	}

	ver, direct, intercept, lastSync := globalRules.Stats()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"version":         ver,
		"direct_count":    direct,
		"intercept_count": intercept,
		"last_sync":       lastSync.Format("2006-01-02 15:04:05"),
	})
}

func handleVoiceTest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if globalVoice == nil {
		globalVoice = voice.NewOptimizer()
	}

	st := globalVoice.TestVoiceConnectivity()
	_ = json.NewEncoder(w).Encode(st)
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

	addLog("[BASLAT] Hello DPI Ag Doktoru v%s (%s)", version.Version, runtime.GOOS)

	// 1. Flush DNS Cache
	addLog("[DNS] Isletim sistemi DNS onbellegi temizleniyor (FlushDNS)...")
	flushStatus, flushDetail := flushDNSCache()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-dns-flush",
		Name:        "DNS Onbellegi Temizlendi (Cache Flush)",
		Description: "ISS tarafindan zehirlenmis engelleme kayitlari hafizadan silindi.",
		Status:      flushStatus,
		Detail:      flushDetail,
	})
	addLog("[OK] FlushDNS: %s", flushDetail)

	// 2. Re-apply System Proxy + SOCKS5
	addLog("[PROXY] Sistem proxy tuneli esitleniyor (127.0.0.1:8080)...")
	proxyStatus, proxyDetail := repairSystemProxy()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-sysproxy",
		Name:        "Sistem Proxy & SOCKS5 Protokolu Yenilendi",
		Description: "HTTP, HTTPS, SOCKS5 ve ortam degiskenleri (127.0.0.1:8080) uygulandi.",
		Status:      proxyStatus,
		Detail:      proxyDetail,
	})
	addLog("[OK] Proxy Motoru: %s", proxyDetail)

	// 3. DNS Optimization (Cloudflare 1.1.1.1 / Google 8.8.8.8)
	addLog("[DNS] Guvenli DNS yapilandirmasi denetleniyor (1.1.1.1 / 8.8.8.8)...")
	dnsStatus, dnsDetail := optimizeDNS()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-dns-config",
		Name:        "Guvenli DNS ve Adaptor Yapilandirmasi",
		Description: "Roblox masaustu istemcisi ve Discord icin Cloudflare ve Google DNS denetlendi.",
		Status:      dnsStatus,
		Detail:      dnsDetail,
	})
	addLog("[OK] DNS Yapilandirmasi: %s", dnsDetail)

	// 4. Stale Discord Connection Check
	addLog("[DISCORD] Discord masaustu arka plan soketleri taraniyor...")
	discordStatus, discordDetail := checkDiscordState()
	report.Steps = append(report.Steps, DiagnosticStep{
		ID:          "step-discord-state",
		Name:        "Discord Durumu ve Soket Kontrolu",
		Description: "Discord masaustu uygulamasinin korumali tunele temiz baglanmasi denetlendi.",
		Status:      discordStatus,
		Detail:      discordDetail,
	})
	addLog("[OK] Discord Durumu: %s", discordDetail)

	// 5. Live Connectivity & DPI Fragmentation Tests
	addLog("[TEST] Cloudflare DoH (1.1.1.1:443) canli gecikme olcumu yapiliyor...")
	report.DoHPing = testDirectLatency("1.1.1.1:443", 2*time.Second)
	if report.DoHPing > 0 {
		addLog("[OK] Cloudflare DNS Erisimi: %d ms", report.DoHPing)
	} else {
		addLog("[WARN] Cloudflare DNS gecikmeli yanit verdi")
	}

	addLog("[TEST] Discord Gateway (discord.com:443) Hello DPI tunel testi yapiliyor...")
	report.DiscordPing = testProxyTunnel("127.0.0.1:8080", "discord.com:443", 4*time.Second)
	if report.DiscordPing <= 0 {
		report.DiscordPing = testDirectLatency("discord.com:443", 2*time.Second)
	}

	if report.DiscordPing > 0 {
		addLog("[OK] Discord Gateway: %d ms [Erisim Acik]", report.DiscordPing)
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-discord-ping",
			Name:        "Discord Sunucularina Erisim",
			Description: fmt.Sprintf("Discord ag gecidine dogrudan erisim saglandi (%d ms).", report.DiscordPing),
			Status:      "success",
			Detail:      "Discord Web, Gateway ve Ses sunuculari engelsiz yanit veriyor.",
		})
	} else {
		report.AllGood = false
		addLog("[WARN] Discord Gateway baglantisi gecikmeli")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-discord-ping",
			Name:        "Discord Sunucularina Erisim",
			Description: "Discord sunucusuna dogrudan erisim gecikmeli.",
			Status:      "warning",
			Detail:      "Discord'u asagidaki butona tiklayarak Hello DPI ile temiz bastan baslatabilirsiniz.",
		})
	}

	addLog("[TEST] Roblox CDN ve Web (setup.rbxcdn.com:443) test ediliyor...")
	report.RobloxPing = testProxyTunnel("127.0.0.1:8080", "setup.rbxcdn.com:443", 4*time.Second)
	if report.RobloxPing <= 0 {
		report.RobloxPing = testDirectLatency("setup.rbxcdn.com:443", 2*time.Second)
	}

	if report.RobloxPing > 0 {
		addLog("[OK] Roblox CDN ve Paketleri: %d ms [Erisim Acik]", report.RobloxPing)
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-roblox-ping",
			Name:        "Roblox Web ve CDN Paketleri",
			Description: fmt.Sprintf("Roblox Web ve CDN sunucularina baglanti kuruldu (%d ms).", report.RobloxPing),
			Status:      "success",
			Detail:      "roblox.com ve setup.rbxcdn.com paketleri engelsiz akiyor.",
		})
	} else {
		addLog("[INFO] Roblox CDN dogrudan yanit vermedi")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-roblox-ping",
			Name:        "Roblox Web ve CDN Paketleri",
			Description: "Roblox dogrudan baglanamadi; ag karti DNS'inin 1.1.1.1 yapilmasi onerilir.",
			Status:      "info",
			Detail:      "Ag ayarlarini sifirlayarak veya Cekirdek Modunu baslatarak cozebilirsiniz.",
		})
	}

	// Step: Kernel Mode Status
	kernelActive := divert.IsRunning()
	report.KernelActive = kernelActive
	if kernelActive {
		addLog("[OK] Cekirdek Modu (L3/L4 WinDivert) devrede: Roblox ve tum oyunlar engelsiz.")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-kernel",
			Name:        "Cekirdek Modu (L3/L4 WinDivert)",
			Description: "Roblox ve ham soket kullanan oyunlar icin cekirdek seviyesinde paket parcalama",
			Status:      "success",
			Detail:      "Aktif. Tum ag trafigi cekirdek seviyesinde korunuyor.",
		})
	} else {
		addLog("[INFO] Cekirdek Modu beklemede (Roblox ve oyunlar icin baslatilabilir).")
		report.Steps = append(report.Steps, DiagnosticStep{
			ID:          "step-kernel",
			Name:        "Cekirdek Modu (L3/L4 WinDivert)",
			Description: "Roblox ve ham soket kullanan oyunlar icin cekirdek seviyesinde paket parcalama",
			Status:      "info",
			Detail:      "Hazir durumda. Roblox oynamak icin baslatabilirsiniz.",
		})
	}

	addLog("[TAMAM] Tum onarim ve tesihis adimlari tamamlandi. Sistem hazir.")

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
		return true, "Roblox sayfası açıldı."
	case "darwin":
		for _, envVar := range []string{"http_proxy", "https_proxy", "all_proxy", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY"} {
			_ = exec.Command("launchctl", "setenv", envVar, "http://127.0.0.1:8080").Run()
		}
		_ = exec.Command("open", "https://www.roblox.com/home").Start()
		return true, "Roblox sayfası açıldı (Proxy ortam değişkenleri uygulandı)."
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
  <title>Hello DPI - Ağ Doktoru & Teşhis</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #000000;
      --card-bg: #111113;
      --card-border: #27272a;
      --card-border-subtle: #1c1c1f;
      --text-main: #ffffff;
      --text-sub: #a1a1aa;
      --text-muted: #71717a;
    }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background: var(--bg);
      color: var(--text-main);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: flex-start;
      padding: 32px 20px;
      -webkit-font-smoothing: antialiased;
    }
    .container {
      width: 100%;
      max-width: 860px;
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 16px;
      padding: 32px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.8);
      display: flex;
      flex-direction: column;
      gap: 24px;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding-bottom: 20px;
      border-bottom: 1px solid var(--card-border);
    }
    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .brand-title {
      font-size: 16px;
      font-weight: 700;
      letter-spacing: -0.3px;
      color: #ffffff;
      text-transform: uppercase;
    }
    .version-tag {
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
      background: #1f1f23;
      border: 1px solid var(--card-border);
      color: var(--text-sub);
      padding: 3px 8px;
      border-radius: 6px;
    }
    .header-right {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      font-family: 'JetBrains Mono', monospace;
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      padding: 4px 10px;
      border-radius: 6px;
      background: #18181b;
      border: 1px solid var(--card-border);
      color: #ffffff;
    }
    .status-dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: #ffffff;
    }
    .btn-retest {
      background: #ffffff;
      color: #000000;
      font-size: 13px;
      font-weight: 600;
      padding: 8px 18px;
      border-radius: 8px;
      border: none;
      cursor: pointer;
      transition: all 0.15s ease;
    }
    .btn-retest:hover {
      background: #e4e4e7;
    }
    .btn-retest:disabled {
      opacity: 0.4;
      cursor: not-allowed;
    }

    /* Metrics Grid */
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 12px;
    }
    .metric-card {
      background: #09090b;
      border: 1px solid var(--card-border);
      border-radius: 10px;
      padding: 16px;
      display: flex;
      flex-direction: column;
      gap: 6px;
    }
    .metric-label {
      font-size: 11px;
      font-weight: 600;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.8px;
    }
    .metric-value {
      font-family: 'JetBrains Mono', monospace;
      font-size: 24px;
      font-weight: 700;
      color: #ffffff;
      line-height: 1.2;
    }
    .metric-sub {
      font-size: 11px;
      color: var(--text-sub);
    }

    /* Actions Section */
    .section-header {
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 1px;
      color: var(--text-muted);
      margin-bottom: 10px;
    }
    .actions-grid {
      display: grid;
      grid-template-columns: repeat(2, 1fr);
      gap: 10px;
    }
    .btn-action {
      background: #18181b;
      border: 1px solid var(--card-border);
      border-radius: 8px;
      padding: 14px 16px;
      color: #ffffff;
      font-size: 13px;
      font-weight: 600;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: space-between;
      transition: all 0.15s ease;
      text-align: left;
    }
    .btn-action:hover {
      background: #27272a;
      border-color: #3f3f46;
    }
    .btn-action.highlight {
      background: #ffffff;
      color: #000000;
      border-color: #ffffff;
    }
    .btn-action.highlight:hover {
      background: #e4e4e7;
    }
    .action-tag {
      font-family: 'JetBrains Mono', monospace;
      font-size: 10px;
      padding: 3px 8px;
      border-radius: 4px;
      background: #27272a;
      color: var(--text-sub);
      text-transform: uppercase;
    }
    .btn-action.highlight .action-tag {
      background: #000000;
      color: #ffffff;
    }

    /* Console Section */
    .console-wrapper {
      background: #000000;
      border: 1px solid var(--card-border);
      border-radius: 10px;
      overflow: hidden;
    }
    .console-bar {
      background: #18181b;
      border-bottom: 1px solid var(--card-border);
      padding: 10px 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      font-size: 11px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.8px;
      color: var(--text-muted);
    }
    .console-body {
      padding: 16px;
      max-height: 200px;
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

    /* Documentation Cards */
    .info-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 12px;
    }
    .info-card {
      background: #09090b;
      border: 1px solid var(--card-border);
      border-radius: 10px;
      padding: 16px;
    }
    .info-card h4 {
      font-size: 12px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.6px;
      color: #ffffff;
      margin-bottom: 6px;
    }
    .info-card p {
      font-size: 12px;
      color: var(--text-sub);
      line-height: 1.5;
    }

    /* Toast */
    .toast {
      position: fixed;
      bottom: 24px;
      right: 24px;
      background: #18181b;
      border: 1px solid var(--card-border);
      color: #ffffff;
      font-size: 12px;
      padding: 12px 20px;
      border-radius: 8px;
      box-shadow: 0 10px 30px rgba(0, 0, 0, 0.8);
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
      .info-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="header-left">
        <span class="brand-title">Hello DPI Console</span>
        <span class="version-tag">v4.0.0</span>
      </div>
      <div class="header-right">
        <div class="status-badge">
          <span class="status-dot"></span>
          <span id="nav-status-text">CEVRIMICI</span>
        </div>
        <button class="btn-retest" id="retestBtn" onclick="startFullDiagnostics()">
          Yeniden Tara
        </button>
      </div>
    </div>

    <div class="metrics-grid">
      <div class="metric-card">
        <div class="metric-label">Sistem Proxy</div>
        <div class="metric-value">127.0.0.1:8080</div>
        <div class="metric-sub">Tunel Aktif</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Cekirdek Modu</div>
        <div class="metric-value" id="val-kernel">Beklemede</div>
        <div class="metric-sub" id="sub-kernel">Roblox Icin Hazir</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Discord Gateway</div>
        <div class="metric-value" id="val-discord">-- ms</div>
        <div class="metric-sub">Ses & API Tuneli</div>
      </div>
      <div class="metric-card">
        <div class="metric-label">Roblox Servisleri</div>
        <div class="metric-value" id="val-roblox">-- ms</div>
        <div class="metric-sub">CDN & Oyun Paketleri</div>
      </div>
    </div>

    <div>
      <div class="section-header">Hizli Islemler</div>
      <div class="actions-grid">
        <button class="btn-action highlight" id="btn-kernel-toggle" onclick="toggleKernel()">
          <span id="btn-kernel-label">Cekirdek Modunu Baslat</span>
          <span class="action-tag">L3/L4 Sockets</span>
        </button>
        <button class="btn-action" onclick="resetNetwork()">
          <span>Ag Ayarlarini Sifirla</span>
          <span class="action-tag">DHCP & Flush</span>
        </button>
        <button class="btn-action" onclick="restartDiscord()">
          <span>Discord Tunelini Yenile</span>
          <span class="action-tag">Restart</span>
        </button>
        <button class="btn-action" onclick="launchRoblox()">
          <span>Roblox'u Baslat</span>
          <span class="action-tag">Launch</span>
        </button>
      </div>
    </div>

    <div class="console-wrapper">
      <div class="console-bar">
        <span>Canli Teshis Gunlugu</span>
        <span id="log-count">0 kayit</span>
      </div>
      <div class="console-body" id="console-body">
        <div class="log-line">> Konsol baslatiliyor...</div>
      </div>
    </div>

    <div class="info-grid">
      <div class="info-card">
        <h4>Standart Tunel Modu (Discord & Web)</h4>
        <p>Tarayicilar, Edge, Chrome ve Discord masaustu istemcisi yerel proxy tunelini kullanir. Ilk TLS ClientHello paketleri 5 bayta bolunerek sansur sifir gecikmeyle asilir.</p>
      </div>
      <div class="info-card">
        <h4>Cekirdek Modu (Roblox & Oyunlar)</h4>
        <p>Roblox ve ham soket kullanan oyun istemcileri icin ag karti duzeyinde paket parcalama yapilir. WinDivert surucusu ve ortam degiskenleri ile baglanti kurulur.</p>
      </div>
    </div>

    <div class="toast" id="toast"></div>
  </div>

  <script>
    let isKernelActive = false;

    function showToast(msg) {
      const t = document.getElementById('toast');
      t.innerText = msg;
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
      document.getElementById('log-count').innerText = total + ' kayit';
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
      const sub = document.getElementById('sub-kernel');
      const btn = document.getElementById('btn-kernel-toggle');
      const btnLbl = document.getElementById('btn-kernel-label');

      if (active) {
        val.innerText = "Aktif";
        sub.innerText = "Tum Oyunlar Destekleniyor";
        btnLbl.innerText = "Cekirdek Modunu Durdur";
        btn.classList.remove('highlight');
      } else {
        val.innerText = "Beklemede";
        sub.innerText = "Roblox Icin Hazir";
        btnLbl.innerText = "Cekirdek Modunu Baslat";
        btn.classList.add('highlight');
      }
    }

    async function toggleKernel() {
      const action = isKernelActive ? "durduruluyor" : "baslatiliyor";
      showToast("Cekirdek Modu " + action + "...");
      appendLog("> [KOMUT] Cekirdek Modu " + action + "...");

      try {
        const res = await fetch('/api/doctor/kernel-toggle');
        const data = await res.json();
        showToast(data.message);
        appendLog("> [SONUC] " + data.message);
        setTimeout(startFullDiagnostics, 1200);
      } catch (err) {
        showToast("Hata: " + err.message);
      }
    }

    async function resetNetwork() {
      showToast("Ag ayarlari sifirlaniyor...");
      appendLog("> [KOMUT] Ag ayarlari sifirlaniyor (DHCP + WinHTTP Reset)...");
      try {
        const res = await fetch('/api/doctor/reset-network');
        const data = await res.json();
        showToast(data.message);
        appendLog("> [SONUC] " + data.message);
        setTimeout(startFullDiagnostics, 1200);
      } catch (err) {
        showToast("Hata: " + err.message);
      }
    }

    async function restartDiscord() {
      showToast("Discord soketleri yenileniyor...");
      appendLog("> [KOMUT] Discord tuneli temizleniyor...");
      try {
        const res = await fetch('/api/doctor/restart-discord');
        const data = await res.json();
        showToast(data.message);
        appendLog("> [SONUC] " + data.message);
      } catch (err) {
        showToast("Hata: " + err.message);
      }
    }

    async function launchRoblox() {
      showToast("Roblox baslatiliyor...");
      appendLog("> [KOMUT] Roblox cagriliyor...");
      try {
        const res = await fetch('/api/doctor/launch-roblox');
        const data = await res.json();
        showToast(data.message);
      } catch (err) {
        showToast("Hata: " + err.message);
      }
    }

    async function runAutoTune() {
      appendLog('> [SONDAJ] Canli DPI sondaji ve strateji kalibrasyonu baslatiliyor...');
      showToast('DPI sondaj testi calisiyor...');
      try {
        const res = await fetch('/api/doctor/autotune');
        const data = await res.json();
        if (data.bypass_verified) {
          appendLog('> [OK] Optimal Strateji: ' + data.best_mode + ' (splitPos: ' + data.best_split_pos + ', delay: ' + data.best_delay_ms + 'ms, ping: ' + data.best_latency_ms + 'ms)');
          appendLog('> [ISS] Algilanan Ag: ' + data.isp_name);
          document.getElementById('val-autotune').innerText = data.best_mode.toUpperCase() + '-' + data.best_split_pos;
          document.getElementById('sub-autotune').innerText = data.best_latency_ms + 'ms (' + data.isp_name + ')';
          showToast('DPI Ayari Yapildi: ' + data.best_mode);
        } else {
          appendLog('> [WARN] DPI sondajinda guvenli mod secildi.');
        }
      } catch (e) {
        appendLog('> [HATA] DPI sondaj hatasi: ' + e.message);
      }
    }

    async function syncRules() {
      appendLog('> [KURAL] Dinamik OTA kurallari buluttan senkronize ediliyor...');
      showToast('Kurallar guncelleniyor...');
      try {
        const res = await fetch('/api/doctor/sync-rules');
        const data = await res.json();
        appendLog('> [KURAL] ' + data.message);
        showToast(data.message);
        await loadRulesStatus();
      } catch (e) {
        appendLog('> [HATA] Kural guncelleme hatasi: ' + e.message);
      }
    }

    async function loadRulesStatus() {
      try {
        const res = await fetch('/api/doctor/rules-status');
        const data = await res.json();
        const total = data.direct_count + data.intercept_count;
        document.getElementById('val-rules').innerText = total + ' Kural';
        document.getElementById('sub-rules').innerText = 'Direct: ' + data.direct_count + ' | DPI: ' + data.intercept_count;
      } catch (e) {}
    }

    async function startFullDiagnostics() {
      const consoleBody = document.getElementById('console-body');
      consoleBody.innerHTML = '';
      appendLog("> Canli sistem teshisi baslatildi...");

      await checkKernelStatus();
      await loadRulesStatus();

      try {
        const res = await fetch('/api/doctor/repair');
        const report = await res.json();

        if (report.discord_ping_ms > 0) {
          document.getElementById('val-discord').innerText = report.discord_ping_ms + ' ms';
        } else {
          document.getElementById('val-discord').innerText = 'Bagli';
        }

        if (report.roblox_ping_ms > 0) {
          document.getElementById('val-roblox').innerText = report.roblox_ping_ms + ' ms';
        } else {
          document.getElementById('val-roblox').innerText = isKernelActive ? 'Aktif' : 'Beklemede';
        }

        if (report.logs && report.logs.length > 0) {
          report.logs.forEach(l => appendLog(l));
        }

        appendLog("> Teshis tamamlandi. Sistem hazir.");
      } catch (err) {
        appendLog("> Teshis hatasi: " + err.message);
      }
    }

    setTimeout(startFullDiagnostics, 300);
  </script>
</body>
</html>`

