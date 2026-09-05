package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gogpu/systray"
	"github.com/hellodpi/hellodpi/internal/autostart"
	"github.com/hellodpi/hellodpi/internal/doctor"
	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/icon"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/speedtest"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/updater"
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
	statusItem := menu.Add(fmt.Sprintf("👋 Hello DPI: Aktif (v%s)", version.Version), nil)
	statusItem.SetDisabled(true)

	// 2. Protection Toggle (Instant 0ms UI feedback + Async sysproxy toggle)
	isActive := true
	var toggleMu sync.Mutex
	var toggleItem *systray.MenuItem

	toggleItem = menu.Add("⏸️ Korumayı Duraklat", func() {
		toggleMu.Lock()
		defer toggleMu.Unlock()

		if isActive {
			// Pause protection: Update UI INSTANTLY (0ms)
			isActive = false
			tray.SetTemplateIcon(icon.PausedIconPNG())
			tray.SetIcon(icon.PausedIconPNG())
			tray.SetTooltip("Hello DPI: Duraklatıldı")
			statusItem.SetLabel("⏸️ Hello DPI: Duraklatıldı")
			toggleItem.SetLabel("▶️ Korumayı Başlat")
			tray.ShowNotification(appTitle, "Koruma geçici olarak duraklatıldı.")

			// Execute system proxy restoration asynchronously in background
			go func() {
				_ = sysproxy.ClearSystemProxy()
			}()
		} else {
			// Resume protection: Update UI INSTANTLY (0ms)
			isActive = true
			tray.SetTemplateIcon(icon.ActiveIconPNG())
			tray.SetIcon(icon.ActiveIconPNG())
			tray.SetTooltip("Hello DPI: Aktif (Sansürsüz İnternet)")
			statusItem.SetLabel(fmt.Sprintf("👋 Hello DPI: Aktif (v%s)", version.Version))
			toggleItem.SetLabel("⏸️ Korumayı Duraklat")
			tray.ShowNotification(appTitle, "Hello DPI devrede! Discord ve tüm siteler açık.")

			// Execute system proxy configuration asynchronously in background
			go func() {
				_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
			}()
		}
	})

	menu.AddSeparator()

	// 3. One-Click Network Troubleshooter & Doctor (Discord, Roblox, GSB WiFi)
	menu.Add("🛠️ Ağ Sorunlarını Gider (Otomatik Onar)", func() {
		tray.ShowNotification(appTitle, "Ağ sorunları taranıyor ve otomatik gideriliyor...")
		go func() {
			doctor.OpenDoctor(proxyPort)
		}()
	})

	menu.AddSeparator()

	// 4. Custom Animated Speedtest
	menu.Add("⚡ Hız Testi Yap (Speedtest)", func() {
		speedtest.OpenSpeedtest(proxyPort)
		tray.ShowNotification(appTitle, "Özel Hız Testi paneli tarayıcınızda açıldı.")
	})

	menu.AddSeparator()

	// 4. Auto-Updater Action
	var latestRelease *updater.ReleaseInfo
	var updateMu sync.Mutex
	isUpdating := false

	var updateItem *systray.MenuItem
	updateItem = menu.Add("🔄 Güncellemeleri Denetle", func() {
		updateMu.Lock()
		defer updateMu.Unlock()

		if isUpdating {
			return
		}

		if latestRelease != nil && latestRelease.TargetAsset != nil {
			// Apply update
			isUpdating = true
			updateItem.SetLabel("⏳ İndiriliyor... (%0)")
			tray.ShowNotification(appTitle, fmt.Sprintf("%s güncellemesi indiriliyor...", latestRelease.TagName))

			go func() {
				err := updater.ApplyUpdate(latestRelease, func(percent int) {
					updateItem.SetLabel(fmt.Sprintf("⏳ İndiriliyor... (%%%d)", percent))
				})

				if err != nil {
					updateMu.Lock()
					isUpdating = false
					updateMu.Unlock()
					updateItem.SetLabel("❌ Güncelleme Başarısız")
					tray.ShowNotification(appTitle, "Güncelleme hatası: "+err.Error())
					return
				}

				tray.ShowNotification(appTitle, "Güncelleme tamamlandı! Yeniden başlatılıyor...")
				time.Sleep(1 * time.Second)
				_ = updater.RestartApp()
			}()
			return
		}

		// Manual check
		updateItem.SetLabel("⏳ Denetleniyor...")
		go func() {
			rel, isNew, err := updater.CheckUpdate()
			updateMu.Lock()
			defer updateMu.Unlock()

			if err != nil {
				updateItem.SetLabel("🔄 Güncellemeleri Denetle")
				tray.ShowNotification(appTitle, "Güncelleme denetlenemedi: "+err.Error())
				return
			}

			if isNew && rel != nil && rel.TargetAsset != nil {
				latestRelease = rel
				updateItem.SetLabel(fmt.Sprintf("✨ Yeni Güncelleme: %s (Tıkla ve Güncelle)", rel.TagName))
				tray.ShowNotification(appTitle, fmt.Sprintf("Yeni sürüm mevcut: %s! Güncellemek için tıklayın.", rel.TagName))
			} else {
				updateItem.SetLabel("✓ En Son Sürüm Kullanılıyor")
				tray.ShowNotification(appTitle, fmt.Sprintf("Hello DPI güncel! (v%s)", version.Version))
				time.AfterFunc(10*time.Second, func() {
					updateMu.Lock()
					if latestRelease == nil {
						updateItem.SetLabel("🔄 Güncellemeleri Denetle")
					}
					updateMu.Unlock()
				})
			}
		}()
	})

	menu.AddSeparator()

	// 5. Launch on Boot Toggle (Persistence)
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

	// 6. Quit
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

	// Periodic background check for updates (3 seconds after startup, then every 4 hours)
	go func() {
		time.Sleep(3 * time.Second)
		for {
			rel, isNew, err := updater.CheckUpdate()
			if err == nil && isNew && rel != nil && rel.TargetAsset != nil {
				updateMu.Lock()
				latestRelease = rel
				updateItem.SetLabel(fmt.Sprintf("✨ Yeni Güncelleme: %s (Tıkla ve Güncelle)", rel.TagName))
				updateMu.Unlock()
				tray.ShowNotification(appTitle, fmt.Sprintf("✨ Yeni sürüm yayınlandı: %s! Güncellemek için menüye tıklayın.", rel.TagName))
				break
			}
			time.Sleep(4 * time.Hour)
		}
	}()

	// Notify user on launch
	tray.Show()
	tray.ShowNotification(appTitle, "Hello DPI v"+version.Version+" devrede. Menü çubuğundan kontrol edebilirsiniz.")

	// Start tray event loop
	if err := tray.Run(); err != nil {
		log.Printf("[Hello DPI Tray] Error running tray: %v", err)
	}
}
