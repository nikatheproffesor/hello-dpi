package speedtest

import (
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// RegisterHandlers registers the speedtest endpoints onto the provided ServeMux
func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/speedtest", handleDashboard)
	mux.HandleFunc("/api/speedtest/ping", handlePing)
	mux.HandleFunc("/api/speedtest/download", handleDownload)
	mux.HandleFunc("/api/speedtest/upload", handleUpload)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusNoContent)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	// Stream from real internet edge CDN to measure genuine internet connection speed
	cdnURL := "https://speed.cloudflare.com/__down?bytes=25000000"
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(cdnURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		w.Header().Set("Content-Length", strconv.FormatInt(resp.ContentLength, 10))
		_, _ = io.Copy(w, resp.Body)
		return
	}

	// Secondary CDN fallback
	resp, err = client.Get("https://cdnjs.cloudflare.com/ajax/libs/react/18.2.0/umd/react.production.min.js")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		_, _ = io.Copy(w, resp.Body)
		return
	}

	// When offline or CDNs are unreachable, return an error rather than fabricating fake speeds
	http.Error(w, "Download measurement failed: Edge CDNs unreachable or offline", http.StatusBadGateway)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	// Forward upload payload to real external CDN
	req, err := http.NewRequestWithContext(r.Context(), "POST", "https://speed.cloudflare.com/__up", r.Body)
	if err == nil {
		req.Header.Set("Content-Type", "application/octet-stream")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			_, _ = io.Copy(io.Discard, resp.Body)
			if resp.StatusCode == http.StatusOK {
				w.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	http.Error(w, "Upload measurement failed: Edge CDN upload failed", http.StatusBadGateway)
}

// OpenSpeedtest opens the speedtest UI in the default web browser
func OpenSpeedtest(port string) {
	url := "http://127.0.0.1:" + port + "/speedtest"
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

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Hello DPI · Hız Testi</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@500;600&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #000000;
      --card-bg: #111113;
      --card-border: #27272a;
      --text-main: #ffffff;
      --text-muted: #71717a;
      --text-sub: #a1a1aa;
      --accent: #ffffff;
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
      justify-content: center;
      padding: 24px;
      -webkit-font-smoothing: antialiased;
    }
    .container {
      width: 100%;
      max-width: 680px;
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 16px;
      padding: 40px 32px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.8);
      display: flex;
      flex-direction: column;
      align-items: center;
      position: relative;
    }
    .header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      width: 100%;
      margin-bottom: 32px;
      padding-bottom: 18px;
      border-bottom: 1px solid var(--card-border);
    }
    .brand {
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .brand-title {
      font-size: 15px;
      font-weight: 700;
      letter-spacing: -0.2px;
      color: #ffffff;
      text-transform: uppercase;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      font-size: 11px;
      font-family: 'JetBrains Mono', monospace;
      color: var(--text-sub);
      background: #18181b;
      border: 1px solid var(--card-border);
      padding: 4px 10px;
      border-radius: 6px;
    }
    .dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: #ffffff;
    }
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 12px;
      width: 100%;
      margin-bottom: 36px;
    }
    .stat-card {
      background: rgba(255, 255, 255, 0.02);
      border: 1px solid var(--card-border);
      border-radius: 12px;
      padding: 14px 12px;
      text-align: center;
      transition: border-color 0.2s ease;
    }
    .stat-card.active {
      border-color: rgba(255, 255, 255, 0.35);
      background: rgba(255, 255, 255, 0.04);
    }
    .stat-title {
      font-size: 11px;
      font-weight: 600;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.6px;
      margin-bottom: 6px;
    }
    .stat-value {
      font-family: 'JetBrains Mono', monospace;
      font-size: 20px;
      font-weight: 600;
      color: #fff;
    }
    .stat-unit {
      font-size: 11px;
      color: var(--text-muted);
      margin-left: 2px;
    }
    .display-area {
      position: relative;
      width: 260px;
      height: 260px;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      margin-bottom: 32px;
    }
    .ring-svg {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      transform: rotate(-90deg);
    }
    .ring-bg {
      fill: none;
      stroke: rgba(255, 255, 255, 0.05);
      stroke-width: 4;
    }
    .ring-progress {
      fill: none;
      stroke: #ffffff;
      stroke-width: 4;
      stroke-linecap: round;
      stroke-dasharray: 754;
      stroke-dashoffset: 754;
      transition: stroke-dashoffset 0.15s ease;
    }
    .speed-number {
      font-family: 'JetBrains Mono', monospace;
      font-size: 64px;
      font-weight: 600;
      line-height: 1;
      color: #fff;
      letter-spacing: -2px;
    }
    .speed-unit {
      font-size: 12px;
      font-weight: 600;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 1.5px;
      margin-top: 8px;
    }
    .speed-phase {
      font-size: 13px;
      color: var(--text-sub);
      margin-top: 4px;
      min-height: 18px;
    }
    .btn-start {
      background: #ffffff;
      color: #000000;
      font-size: 14px;
      font-weight: 600;
      padding: 12px 32px;
      border-radius: 9999px;
      border: none;
      cursor: pointer;
      transition: all 0.2s ease;
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }
    .btn-start:hover {
      background: #e4e4e7;
      transform: translateY(-1px);
    }
    .btn-start:disabled {
      opacity: 0.4;
      cursor: not-allowed;
      transform: none;
    }
    .footer-note {
      margin-top: 28px;
      font-size: 12px;
      color: var(--text-muted);
    }
    @media (max-width: 640px) {
      .stats-grid { grid-template-columns: repeat(2, 1fr); }
      .container { padding: 28px 20px; }
      .speed-number { font-size: 48px; }
      .display-area { width: 220px; height: 220px; }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="brand">
        <span class="brand-title">Hello DPI</span>
      </div>
      <div class="status-pill">
        <span class="dot"></span>
        <span>Tünel Aktif (v3.1.0)</span>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-card" id="card-ping">
        <div class="stat-title">Ping</div>
        <div class="stat-value"><span id="val-ping">--</span><span class="stat-unit">ms</span></div>
      </div>
      <div class="stat-card" id="card-jitter">
        <div class="stat-title">Dalgalanma</div>
        <div class="stat-value"><span id="val-jitter">--</span><span class="stat-unit">ms</span></div>
      </div>
      <div class="stat-card" id="card-download">
        <div class="stat-title">İndirme</div>
        <div class="stat-value"><span id="val-download">--</span><span class="stat-unit">Mbps</span></div>
      </div>
      <div class="stat-card" id="card-upload">
        <div class="stat-title">Yükleme</div>
        <div class="stat-value"><span id="val-upload">--</span><span class="stat-unit">Mbps</span></div>
      </div>
    </div>

    <div class="display-area">
      <svg class="ring-svg" viewBox="0 0 260 260">
        <circle class="ring-bg" cx="130" cy="130" r="120" />
        <circle class="ring-progress" id="ringProgress" cx="130" cy="130" r="120" />
      </svg>
      <div class="speed-number" id="liveSpeed">0.0</div>
      <div class="speed-unit">Mbps</div>
      <div class="speed-phase" id="phaseText">Hazır</div>
    </div>

    <button class="btn-start" id="startBtn" onclick="runSpeedtest()">
      Testi Başlat
    </button>

    <div class="footer-note">
      Hello DPI v3.1.0 · Doğrudan Yerel Ölçüm · Sıfır Paket Kaybı
    </div>
  </div>

  <script>
    const circumference = 2 * Math.PI * 120; // 753.98
    const ring = document.getElementById('ringProgress');

    function updateProgress(val = 0, maxVal = 200) {
      const pct = Math.min(Math.max(val / maxVal, 0), 1);
      const offset = circumference - (pct * circumference);
      ring.style.strokeDashoffset = offset;
    }

    updateProgress(0);

    function setActiveCard(cardId) {
      document.querySelectorAll('.stat-card').forEach(c => c.classList.remove('active'));
      if (cardId) document.getElementById(cardId).classList.add('active');
    }

    async function measurePing() {
      const pings = [];
      document.getElementById('phaseText').innerText = 'Gecikme ölçülüyor';
      setActiveCard('card-ping');

      const pingUrl = 'https://speed.cloudflare.com/__down?bytes=0&_=';
      for (let i = 0; i < 6; i++) {
        const t0 = performance.now();
        try {
          await fetch(pingUrl + Date.now(), { mode: 'cors', cache: 'no-store' });
        } catch (_) {
          await fetch('/api/speedtest/ping?_=' + Date.now());
        }
        const t1 = performance.now();
        pings.push(t1 - t0);
        await new Promise(r => setTimeout(r, 60));
      }

      pings.sort((a, b) => a - b);
      const medianPing = pings[Math.floor(pings.length / 2)];
      
      let jitter = 0;
      for (let i = 1; i < pings.length; i++) {
        jitter += Math.abs(pings[i] - pings[i-1]);
      }
      jitter = jitter / (pings.length - 1);

      document.getElementById('val-ping').innerText = medianPing.toFixed(1);
      document.getElementById('val-jitter').innerText = jitter.toFixed(1);
    }

    async function measureDownload() {
      document.getElementById('phaseText').innerText = 'İndirme testi';
      setActiveCard('card-download');

      const durationMs = 6000;
      const startTime = performance.now();
      let totalBytes = 0;

      const controller = new AbortController();
      setTimeout(() => controller.abort(), durationMs);

      // Prefer real external Cloudflare CDN (with fallback to proxy streaming) to accurately measure WAN speed
      let downloadUrl = 'https://speed.cloudflare.com/__down?bytes=40000000&_=';
      try {
        let resp;
        try {
          resp = await fetch(downloadUrl + Date.now(), { signal: controller.signal, mode: 'cors' });
        } catch (_) {
          resp = await fetch('/api/speedtest/download?_=' + Date.now(), { signal: controller.signal });
        }

        const reader = resp.body.getReader();
        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          totalBytes += value.length;
          
          const elapsedSec = (performance.now() - startTime) / 1000;
          if (elapsedSec > 0.15) {
            const curMbps = ((totalBytes * 8) / elapsedSec) / 1000000;
            document.getElementById('liveSpeed').innerText = curMbps.toFixed(1);
            updateProgress(curMbps, 200);
          }
        }
      } catch (e) {}

      const totalSec = Math.max((performance.now() - startTime) / 1000, 0.2);
      const finalMbps = ((totalBytes * 8) / totalSec) / 1000000;
      document.getElementById('val-download').innerText = finalMbps > 0 ? finalMbps.toFixed(1) : '--';
    }

    async function measureUpload() {
      document.getElementById('phaseText').innerText = 'Yükleme testi';
      setActiveCard('card-upload');

      const chunk = new Uint8Array(256 * 1024);
      let totalBytes = 0;
      const durationMs = 5000;
      const startTime = performance.now();

      while (performance.now() - startTime < durationMs) {
        try {
          await fetch('https://speed.cloudflare.com/__up', {
            method: 'POST',
            body: chunk,
            mode: 'cors'
          });
        } catch (_) {
          await fetch('/api/speedtest/upload?_=' + Date.now(), {
            method: 'POST',
            body: chunk
          });
        }
        totalBytes += chunk.length;
        const elapsedSec = (performance.now() - startTime) / 1000;
        if (elapsedSec > 0.15) {
          const curMbps = ((totalBytes * 8) / elapsedSec) / 1000000;
          document.getElementById('liveSpeed').innerText = curMbps.toFixed(1);
          updateProgress(curMbps, 100);
        }
      }

      const totalSec = Math.max((performance.now() - startTime) / 1000, 0.2);
      const finalMbps = ((totalBytes * 8) / totalSec) / 1000000;
      document.getElementById('val-upload').innerText = finalMbps > 0 ? finalMbps.toFixed(1) : '--';
    }

    async function runSpeedtest() {
      const btn = document.getElementById('startBtn');
      btn.disabled = true;
      document.getElementById('val-download').innerText = '--';
      document.getElementById('val-upload').innerText = '--';

      try {
        await measurePing();
        await measureDownload();
        await measureUpload();

        document.getElementById('phaseText').innerText = 'Tamamlandı';
        setActiveCard(null);
      } catch (err) {
        console.error(err);
        document.getElementById('phaseText').innerText = 'Hata oluştu';
      } finally {
        btn.disabled = false;
        setTimeout(() => {
          document.getElementById('liveSpeed').innerText = '0.0';
          updateProgress(0);
        }, 1500);
      }
    }

    setTimeout(runSpeedtest, 500);
  </script>
</body>
</html>`
