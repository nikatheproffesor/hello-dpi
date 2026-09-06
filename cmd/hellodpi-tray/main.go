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
	"github.com/hellodpi/hellodpi/internal/divert"
	"github.com/hellodpi/hellodpi/internal/doctor"
	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/icon"
	"github.com/hellodpi/hellodpi/internal/netmon"
	"github.com/hellodpi/hellodpi/internal/probe"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/speedtest"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/updater"
	"github.com/hellodpi/hellodpi/internal/version"
	"github.com/hellodpi/hellodpi/internal/voice"
)

const (
	proxyPort = "8080"
	proxyAddr = "127.0.0.1:" + proxyPort
	appTitle  = "Hello DPI"
)

func main() {
	sysproxy.RegisterExitCleanup()
	defer sysproxy.RecoverAndClear()

	defer func() {
		_ = divert.Stop()
		_ = sysproxy.ClearSystemProxy()
	}()

	// Initialize Auto-Tune probe engine and WebRTC voice optimizer
	probeEngine := probe.NewEngine()
	voiceOptimizer := voice.NewOptimizer()

	// Load persisted auto-tune profile from disk for instant zero-wait startup
	splitMode := dpi.SplitAuto
	delayMs := 5
	if tuned := probeEngine.GetLastResult(); tuned != nil && tuned.BestStrategy != "" {
		splitMode = dpi.SplitMode(tuned.BestStrategy)
		if tuned.BestDelayMs > 0 {
			delayMs = tuned.BestDelayMs
		}
		log.Printf("[Hello DPI Tray] Loaded persisted tuning profile: strategy=%s, delay=%dms (ISP: %s)", tuned.BestStrategy, delayMs, tuned.ISPName)
	}

	// Initialize core proxy server
	cfg := proxy.Config{
		Addr:        proxyAddr,
		SplitMode:   splitMode,
		DelayMs:     delayMs,
		DoHEndpoint: string(doh.Cloudflare),
		EnableDoH:   true,
	}
	server := proxy.NewServer(cfg)

	doctor.SetCoreEngines(server.Rules, probeEngine, voiceOptimizer, func(mode dpi.SplitMode, splitOffset int, delayMs int) {
		server.UpdateEngineConfig(mode, splitOffset, delayMs)
	})

	// Start proxy server in background
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("[Hello DPI Tray] Proxy server error: %v", err)
		}
	}()

	// Periodic background auto-tune re-verification (every 6 hours)
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			res := probeEngine.RunProbe()
			if res.BypassVerified {
				server.UpdateEngineConfig(dpi.SplitMode(res.BestStrategy), res.BestSplitPos, res.BestDelayMs)
				log.Printf("[Hello DPI Tray] Periodic auto-tune verified: strategy=%s, rtt=%dms", res.BestStrategy, res.BestLatencyMs)
			}
		}
	}()

	// Activate system proxy
	_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)

	// WinDivert operates selectively: L7 is default (zero admin prompt);
	// Kernel divert engages silently only when needed by direct socket games (Roblox)

	// Setup tray
	tray := systray.New()
	tray.SetAppName(appTitle)
	tray.SetTooltip("Hello DPI: Aktif (v" + version.Version + ")")
	tray.SetTemplateIcon(icon.ActiveIconPNG())
	tray.SetIcon(icon.ActiveIconPNG())

	menu := systray.NewMenu()

	// 1. Status Label
	statusItem := menu.Add(fmt.Sprintf("Hello DPI: Aktif (v%s)", version.Version), nil)
	statusItem.SetDisabled(true)

	// 2. Protection Toggle (Instant 0ms UI feedback + Async sysproxy toggle)
	isActive := true
	var toggleMu sync.Mutex
	var toggleItem *systray.MenuItem
	var kernelItem *systray.MenuItem

	toggleItem = menu.Add("Korumayı Duraklat", func() {
		toggleMu.Lock()
		defer toggleMu.Unlock()

		if isActive {
			// Pause protection
			isActive = false
			tray.SetTemplateIcon(icon.PausedIconPNG())
			tray.SetIcon(icon.PausedIconPNG())
			tray.SetTooltip("Hello DPI: Duraklatıldı")
			statusItem.SetLabel("Hello DPI: Duraklatıldı")
			toggleItem.SetLabel("Korumayı Başlat")
			if kernelItem != nil {
				kernelItem.SetLabel("Çekirdek Motoru: Devre Dışı")
			}
			tray.ShowNotification(appTitle, "Koruma geçici olarak duraklatıldı.")

			go func() {
				_ = divert.Stop()
				_ = sysproxy.ClearSystemProxy()
			}()
		} else {
			// Resume protection
			isActive = true
			tray.SetTemplateIcon(icon.ActiveIconPNG())
			tray.SetIcon(icon.ActiveIconPNG())
			tray.SetTooltip("Hello DPI: Aktif (v" + version.Version + ")")
			statusItem.SetLabel(fmt.Sprintf("Hello DPI: Aktif (v%s)", version.Version))
			toggleItem.SetLabel("Korumayı Duraklat")
			if kernelItem != nil {
				kernelItem.SetLabel("Çekirdek Motoru: Aktif")
			}
			tray.ShowNotification(appTitle, "Hello DPI devrede. Discord ve tüm siteler açık.")

			go func() {
				_ = sysproxy.SetSystemProxy("127.0.0.1", 8080)
				_ = divert.Start()
			}()
		}
	})

	menu.AddSeparator()

	// 3. Kernel Divert Engine
	kernelItem = menu.Add("Çekirdek Motoru: Aktif", func() {
		if divert.IsRunning() {
			_ = divert.Stop()
			kernelItem.SetLabel("Çekirdek Motoru: Devre Dışı")
			tray.ShowNotification(appTitle, "Çekirdek Motoru durduruldu.")
		} else {
			err := divert.Start()
			if err != nil {
				tray.ShowNotification(appTitle, "Çekirdek Motoru başlatılamadı: "+err.Error())
			} else {
				kernelItem.SetLabel("Çekirdek Motoru: Aktif")
				tray.ShowNotification(appTitle, "Çekirdek Motoru devrede.")
			}
		}
	})

	// Ensure kernel item label matches actual running state on startup
	go func() {
		_ = divert.Start()
		time.Sleep(500 * time.Millisecond)
		if divert.IsRunning() {
			kernelItem.SetLabel("Çekirdek Motoru: Aktif")
		} else {
			kernelItem.SetLabel("Çekirdek Motoru")
		}
	}()

	menu.AddSeparator()

	// 4. One-Click Network Troubleshooter & Doctor
	menu.Add("Ağ Doktoru & Teşhis", func() {
		go func() {
			doctor.OpenDoctor(proxyPort)
		}()
	})

	// 5. Custom Minimalist Speedtest
	menu.Add("Hız Testi", func() {
		speedtest.OpenSpeedtest(proxyPort)
	})

	menu.AddSeparator()

	// Silent background auto-tuning and dynamic rules sync
	go func() {
		_ = server.Rules.SyncRemote("")
		res := probeEngine.RunProbe()
		if res.BypassVerified {
			server.UpdateEngineConfig(dpi.SplitMode(res.BestMode), res.BestSplitPos, res.BestDelayMs)
			server.Orchestrator.UpdateGroupStrategies(res.GroupStrategies, res.GroupFallbacks)
		}
	}()

	// Periodic silent re-probe every 15 minutes
	probeEngine.StartPeriodicReProbe(15*time.Minute, func(res *probe.Result) {
		if res.BypassVerified {
			server.UpdateEngineConfig(dpi.SplitMode(res.BestMode), res.BestSplitPos, res.BestDelayMs)
			server.Orchestrator.UpdateGroupStrategies(res.GroupStrategies, res.GroupFallbacks)
		}
	})

	// Network interface and sleep/wake monitor
	netMonitor := netmon.NewMonitor("127.0.0.1", 8080, func(oldState, newState netmon.NetworkState) {
		probeEngine.TriggerImmediateReProbe()
	})
	netMonitor.SetProxyState(true)
	netMonitor.Start()
	defer netMonitor.Stop()

	// 6. Auto-Updater Action
	var latestRelease *updater.ReleaseInfo
	var updateMu sync.Mutex
	isUpdating := false

	var updateItem *systray.MenuItem
	updateItem = menu.Add("Güncellemeleri Denetle", func() {
		updateMu.Lock()
		defer updateMu.Unlock()

		if isUpdating {
			return
		}

		if latestRelease != nil && latestRelease.TargetAsset != nil {
			// Apply update
			isUpdating = true
			updateItem.SetLabel("İndiriliyor... (%0)")
			tray.ShowNotification(appTitle, fmt.Sprintf("%s güncellemesi indiriliyor...", latestRelease.TagName))

			go func() {
				err := updater.ApplyUpdate(latestRelease, func(percent int) {
					updateItem.SetLabel(fmt.Sprintf("İndiriliyor... (%%%d)", percent))
				})

				if err != nil {
					updateMu.Lock()
					isUpdating = false
					updateMu.Unlock()
					updateItem.SetLabel("Güncelleme Başarısız")
					tray.ShowNotification(appTitle, "Güncelleme hatası: "+err.Error())
					return
				}

				tray.ShowNotification(appTitle, "Güncelleme tamamlandı. Yeniden başlatılıyor...")
				time.Sleep(1 * time.Second)
				_ = updater.RestartApp()
			}()
			return
		}

		// Manual check
		updateItem.SetLabel("Denetleniyor...")
		go func() {
			rel, isNew, err := updater.CheckUpdate()
			updateMu.Lock()
			defer updateMu.Unlock()

			if err != nil {
				updateItem.SetLabel("Güncellemeleri Denetle")
				tray.ShowNotification(appTitle, "Güncelleme denetlenemedi: "+err.Error())
				return
			}

			if isNew && rel != nil && rel.TargetAsset != nil {
				latestRelease = rel
				updateItem.SetLabel(fmt.Sprintf("Yeni Güncelleme: %s (Tıkla ve Güncelle)", rel.TagName))
				tray.ShowNotification(appTitle, fmt.Sprintf("Yeni sürüm mevcut: %s. Güncellemek için tıklayın.", rel.TagName))
			} else {
				updateItem.SetLabel("En Son Sürüm Kullanılıyor")
				tray.ShowNotification(appTitle, fmt.Sprintf("Hello DPI güncel (v%s)", version.Version))
				time.AfterFunc(10*time.Second, func() {
					updateMu.Lock()
					if latestRelease == nil {
						updateItem.SetLabel("Güncellemeleri Denetle")
					}
					updateMu.Unlock()
				})
			}
		}()
	})

	menu.AddSeparator()

	// 7. Launch on Boot Toggle (Persistence)
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

	// 8. Quit
	menu.Add("Çıkış", func() {
		_ = divert.Stop()
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
		_ = divert.Stop()
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
				updateItem.SetLabel(fmt.Sprintf("Yeni Güncelleme: %s (Tıkla ve Güncelle)", rel.TagName))
				updateMu.Unlock()
				tray.ShowNotification(appTitle, fmt.Sprintf("Yeni sürüm yayınlandı: %s. Güncellemek için menüye tıklayın.", rel.TagName))
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
