package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gogpu/systray"
	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/icon"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
)

const (
	proxyAddr = "127.0.0.1:8080"
	appTitle  = "Hello DPI"
	version   = "1.1.0"
)

func main() {
	// Initialize core proxy server
	cfg := proxy.Config{
		Addr:        proxyAddr,
		SplitMode:   dpi.SplitTLS,
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
	tray.SetIcon(icon.ActiveIconPNG())

	menu := systray.NewMenu()

	// 1. Status Label
	statusItem := menu.Add("🛡️ Hello DPI: Aktif", nil)
	statusItem.SetDisabled(true)

	// 2. Protection Toggle
	isActive := true
	var toggleItem *systray.MenuItem
	toggleItem = menu.Add("⏸️ Korumayı Duraklat", func() {
		if isActive {
			// Pause protection
			_ = sysproxy.ClearSystemProxy()
			isActive = false
			tray.SetIcon(icon.PausedIconPNG())
			tray.SetTooltip("Hello DPI: Duraklatıldı")
			statusItem.SetLabel("⏸️ Hello DPI: Duraklatıldı")
			toggleItem.SetLabel("▶️ Korumayı Başlat")
			tray.ShowNotification(appTitle, "Koruma geçici olarak duraklatıldı.")
		} else {
			// Resume protection
			_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
			isActive = true
			tray.SetIcon(icon.ActiveIconPNG())
			tray.SetTooltip("Hello DPI: Aktif (Sansürsüz İnternet)")
			statusItem.SetLabel("🛡️ Hello DPI: Aktif")
			toggleItem.SetLabel("⏸️ Korumayı Duraklat")
			tray.ShowNotification(appTitle, "Hello DPI devrede! Discord ve tüm siteler açık.")
		}
	})

	menu.AddSeparator()

	// 3. Quick Connection Test
	menu.Add("⚡ Bağlantıyı Test Et", func() {
		go func() {
			start := time.Now()
			proxyURL, _ := url.Parse("http://127.0.0.1:8080")
			client := &http.Client{
				Timeout: 4 * time.Second,
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()

			req, _ := http.NewRequestWithContext(ctx, "GET", "https://discord.com", nil)
			resp, err := client.Do(req)
			elapsed := time.Since(start)

			if err == nil && resp.StatusCode == 200 {
				tray.ShowNotification(appTitle, fmt.Sprintf("✓ Bağlantı Başarılı!\nDiscord ve web siteleri açık (%dms)", elapsed.Milliseconds()))
			} else {
				tray.ShowNotification(appTitle, "⚠️ Test bağlantısı zaman aşımına uğradı.")
			}
		}()
	})

	menu.AddSeparator()

	// 4. Version Info
	verItem := menu.Add(fmt.Sprintf("ℹ️ Hello DPI v%s", version), nil)
	verItem.SetDisabled(true)

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
	tray.ShowNotification(appTitle, "Hello DPI başlatıldı. Menü çubuğundan kontrol edebilirsiniz.")

	// Start tray event loop
	if err := tray.Run(); err != nil {
		log.Printf("[Hello DPI Tray] Error running tray: %v", err)
	}
}
