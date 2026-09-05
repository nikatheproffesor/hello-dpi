package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogpu/systray"
	"github.com/hellodpi/hellodpi/internal/autostart"
	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/icon"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/speedtest"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/version"
)

const (
	proxyPort = "8080"
	proxyAddr = "127.0.0.1:" + proxyPort
	appTitle  = "Hello DPI"
)

func main() {
	// Initialize core proxy server
	cfg := proxy.Config{
		Addr:        proxyAddr,
		SplitMode:   dpi.SplitAuto,
		DelayMs:     5,
		DoHEndpoint: string(doh.Cloudflare),
		EnableDoH:   true,
	}
	server := proxy.NewServer(cfg)

	// Start proxy server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("[Hello DPI Tray] Proxy server error: %v", err)
		}
	}()

	// Activate system proxy
	_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)

	// Setup tray
	tray := systray.New()
	tray.SetAppName(appTitle)
	tray.SetTooltip("Hello DPI: Aktif (Sansürsüz İnternet)")
	tray.SetTemplateIcon(icon.ActiveIconPNG())
	tray.SetIcon(icon.ActiveIconPNG())

	menu := systray.NewMenu()

	// 1. Status Label
	statusItem := menu.Add("👋 Hello DPI: Aktif", nil)
	statusItem.SetDisabled(true)

	// 2. Protection Toggle
	isActive := true
	var toggleItem *systray.MenuItem
	toggleItem = menu.Add("⏸️ Korumayı Duraklat", func() {
		if isActive {
			// Pause protection
			_ = sysproxy.ClearSystemProxy()
			isActive = false
			tray.SetTemplateIcon(icon.PausedIconPNG())
			tray.SetIcon(icon.PausedIconPNG())
			tray.SetTooltip("Hello DPI: Duraklatıldı")
			statusItem.SetLabel("⏸️ Hello DPI: Duraklatıldı")
			toggleItem.SetLabel("▶️ Korumayı Başlat")
			tray.ShowNotification(appTitle, "Koruma geçici olarak duraklatıldı.")
		} else {
			// Resume protection
			_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
			isActive = true
			tray.SetTemplateIcon(icon.ActiveIconPNG())
			tray.SetIcon(icon.ActiveIconPNG())
			tray.SetTooltip("Hello DPI: Aktif (Sansürsüz İnternet)")
			statusItem.SetLabel("👋 Hello DPI: Aktif")
			toggleItem.SetLabel("⏸️ Korumayı Duraklat")
			tray.ShowNotification(appTitle, "Hello DPI devrede! Discord ve tüm siteler açık.")
		}
	})

	menu.AddSeparator()

	// 3. Custom Animated Speedtest
	menu.Add("⚡ Hız Testi Yap (Speedtest)", func() {
		speedtest.OpenSpeedtest(proxyPort)
		tray.ShowNotification(appTitle, "Özel Hız Testi paneli tarayıcınızda açıldı.")
	})

	menu.AddSeparator()

	// 4. Launch on Boot Toggle (Persistence)
	isAutoStart := autostart.IsEnabled()
	var autoStartItem *systray.MenuItem
	autoStartItem = menu.AddCheckbox("Açılışta Otomatik Başlat", isAutoStart, func() {
		newChecked := !autoStartItem.IsChecked()
		if newChecked {
			if err := autostart.Enable(); err == nil {
				autoStartItem.SetChecked(true)
				tray.ShowNotification(appTitle, "Hello DPI bilgisayar açıldığında otomatik başlayacak.")
			}
		} else {
			if err := autostart.Disable(); err == nil {
				autoStartItem.SetChecked(false)
				tray.ShowNotification(appTitle, "Açılışta otomatik başlatma kapatıldı.")
			}
		}
	})

	menu.AddSeparator()

	// 5. Quit
	menu.Add("❌ Çıkış", func() {
		_ = sysproxy.ClearSystemProxy()
		_ = server.Close()
		tray.Remove()
		os.Exit(0)
	})

	tray.SetMenu(menu)

	// Graceful shutdown on OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		_ = sysproxy.ClearSystemProxy()
		_ = server.Close()
		tray.Remove()
		os.Exit(0)
	}()

	// Notify user on launch
	tray.Show()
	tray.ShowNotification(appTitle, "Hello DPI v"+version.Version+" devrede. Menü çubuğundan kontrol edebilirsiniz.")

	// Start tray event loop
	if err := tray.Run(); err != nil {
		log.Printf("[Hello DPI Tray] Error running tray: %v", err)
	}
}
