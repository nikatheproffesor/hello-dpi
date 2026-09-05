package speedtest

import (
	"crypto/rand"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
)

// RegisterHandlers registers the speedtest endpoints onto the provided ServeMux
func RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/speedtest", handleDashboard)
	mux.HandleFunc("/api/speedtest/ping", handlePing)
	mux.HandleFunc("/api/speedtest/download", handleDownload)
	mux.HandleFunc("/api/speedtest/upload", handleUpload)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusNoContent)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	sizeMB := 15
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 50 {
			sizeMB = v
		}
	}

	totalBytes := int64(sizeMB * 1024 * 1024)
	w.Header().Set("Content-Length", strconv.FormatInt(totalBytes, 10))

	chunk := make([]byte, 64*1024)
	_, _ = rand.Read(chunk)

	written := int64(0)
	for written < totalBytes {
		toWrite := int64(len(chunk))
		if totalBytes-written < toWrite {
			toWrite = totalBytes - written
		}
		n, err := w.Write(chunk[:toWrite])
		if err != nil {
			break
		}
		written += int64(n)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	_, _ = io.Copy(io.Discard, r.Body)
	_ = r.Body.Close()
	w.WriteHeader(http.StatusOK)
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
  <title>Hello DPI - İnternet Hız Testi</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;600;700;800&family=JetBrains+Mono:wght@500;700&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #090b10;
      --card-bg: rgba(18, 24, 38, 0.7);
      --card-border: rgba(255, 255, 255, 0.08);
      --accent-cyan: #00f2fe;
      --accent-purple: #9d4edd;
      --accent-glow: rgba(0, 242, 254, 0.25);
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
      justify-content: center;
      padding: 24px;
      overflow-x: hidden;
      background-image: radial-gradient(circle at 50% 0%, rgba(157, 78, 221, 0.15) 0%, transparent 60%),
                        radial-gradient(circle at 50% 100%, rgba(0, 242, 254, 0.1) 0%, transparent 60%);
    }
    .container {
      width: 100%;
      max-width: 780px;
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 28px;
      padding: 40px;
      backdrop-filter: blur(24px);
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6);
      display: flex;
      flex-direction: column;
      align-items: center;
      position: relative;
    }
    .header {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 28px;
    }
    .logo-badge {
      font-size: 28px;
    }
    h1 {
      font-size: 26px;
      font-weight: 800;
      letter-spacing: -0.5px;
      background: linear-gradient(135deg, #ffffff 40%, var(--accent-cyan));
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 16px;
      width: 100%;
      margin-bottom: 36px;
    }
    .stat-card {
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid var(--card-border);
      border-radius: 18px;
      padding: 16px;
      text-align: center;
      transition: all 0.3s ease;
    }
    .stat-card.active {
      border-color: var(--accent-cyan);
      box-shadow: 0 0 20px var(--accent-glow);
    }
    .stat-title {
      font-size: 13px;
      font-weight: 600;
      color: var(--text-dim);
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 6px;
    }
    .stat-value {
      font-family: 'JetBrains Mono', monospace;
      font-size: 24px;
      font-weight: 700;
      color: #fff;
    }
    .stat-unit {
      font-size: 12px;
      color: var(--text-dim);
      margin-left: 2px;
    }
    .gauge-wrapper {
      position: relative;
      width: 320px;
      height: 320px;
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 10px 0 30px;
    }
    canvas {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
    }
    .gauge-center {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      z-index: 2;
    }
    .speed-number {
      font-family: 'JetBrains Mono', monospace;
      font-size: 56px;
      font-weight: 800;
      line-height: 1;
      color: #fff;
      text-shadow: 0 0 24px rgba(0, 242, 254, 0.4);
    }
    .speed-unit {
      font-size: 16px;
      font-weight: 600;
      color: var(--accent-cyan);
      letter-spacing: 1px;
      margin-top: 6px;
    }
    .speed-phase {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-dim);
      margin-top: 4px;
    }
    .btn-start {
      background: linear-gradient(135deg, var(--accent-cyan), var(--accent-purple));
      color: #000;
      font-size: 17px;
      font-weight: 700;
      padding: 16px 44px;
      border-radius: 99px;
      border: none;
      cursor: pointer;
      box-shadow: 0 10px 30px rgba(0, 242, 254, 0.3);
      transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1);
      display: flex;
      align-items: center;
      gap: 10px;
    }
    .btn-start:hover {
      transform: translateY(-2px) scale(1.03);
      box-shadow: 0 15px 40px rgba(0, 242, 254, 0.5);
    }
    .btn-start:disabled {
      opacity: 0.5;
      cursor: not-allowed;
      transform: none;
    }
    .status-badge {
      margin-top: 20px;
      font-size: 13px;
      color: var(--text-dim);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #10b981;
      box-shadow: 0 0 10px #10b981;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <span class="logo-badge">👋</span>
      <h1>Hello DPI Hız Testi</h1>
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

    <div class="gauge-wrapper">
      <canvas id="gaugeCanvas" width="640" height="640"></canvas>
      <div class="gauge-center">
        <div class="speed-number" id="liveSpeed">0.0</div>
        <div class="speed-unit">Mbps</div>
        <div class="speed-phase" id="phaseText">Hazır</div>
      </div>
    </div>

    <button class="btn-start" id="startBtn" onclick="runSpeedtest()">
      <span>⚡ Testi Başlat</span>
    </button>

    <div class="status-badge">
      <span class="dot"></span>
      <span>DPI Motoru Aktif: Trafik doğrudan hat hızında (%100) işleniyor</span>
    </div>
  </div>

  <script>
    const canvas = document.getElementById('gaugeCanvas');
    const ctx = canvas.getContext('2d');
    let currentAngle = 0;
    let targetAngle = 0;

    function drawGauge(val = 0, maxVal = 200) {
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      const cx = canvas.width / 2;
      const cy = canvas.height / 2;
      const r = 240;

      // Background Arc
      ctx.beginPath();
      ctx.arc(cx, cy, r, Math.PI * 0.75, Math.PI * 2.25);
      ctx.strokeStyle = 'rgba(255, 255, 255, 0.08)';
      ctx.lineWidth = 20;
      ctx.lineCap = 'round';
      ctx.stroke();

      // Progress Arc
      const pct = Math.min(Math.max(val / maxVal, 0), 1);
      const endAngle = Math.PI * 0.75 + pct * (Math.PI * 1.5);

      if (pct > 0) {
        const grad = ctx.createLinearGradient(0, 0, canvas.width, canvas.height);
        grad.addColorStop(0, '#00f2fe');
        grad.addColorStop(1, '#9d4edd');
        ctx.beginPath();
        ctx.arc(cx, cy, r, Math.PI * 0.75, endAngle);
        ctx.strokeStyle = grad;
        ctx.lineWidth = 20;
        ctx.lineCap = 'round';
        ctx.shadowColor = '#00f2fe';
        ctx.shadowBlur = 18;
        ctx.stroke();
        ctx.shadowBlur = 0;
      }
    }

    drawGauge(0);

    function setActiveCard(cardId) {
      document.querySelectorAll('.stat-card').forEach(c => c.classList.remove('active'));
      if (cardId) document.getElementById(cardId).classList.add('active');
    }

    async function measurePing() {
      const pings = [];
      document.getElementById('phaseText').innerText = 'Gecikme Ölçülüyor...';
      setActiveCard('card-ping');

      for (let i = 0; i < 6; i++) {
        const t0 = performance.now();
        await fetch('/api/speedtest/ping?_=' + Date.now());
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
      document.getElementById('phaseText').innerText = 'İndirme Testi...';
      setActiveCard('card-download');

      const durationMs = 7000;
      const startTime = performance.now();
      let totalBytes = 0;

      const controller = new AbortController();
      setTimeout(() => controller.abort(), durationMs);

      try {
        const resp = await fetch('/api/speedtest/download?size=35&_=' + Date.now(), { signal: controller.signal });
        const reader = resp.body.getReader();

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          totalBytes += value.length;
          
          const elapsedSec = (performance.now() - startTime) / 1000;
          if (elapsedSec > 0.2) {
            const curMbps = ((totalBytes * 8) / elapsedSec) / 1000000;
            document.getElementById('liveSpeed').innerText = curMbps.toFixed(1);
            drawGauge(curMbps, 200);
          }
        }
      } catch (e) {}

      const totalSec = (performance.now() - startTime) / 1000;
      const finalMbps = ((totalBytes * 8) / totalSec) / 1000000;
      document.getElementById('val-download').innerText = finalMbps.toFixed(1);
    }

    async function measureUpload() {
      document.getElementById('phaseText').innerText = 'Yükleme Testi...';
      setActiveCard('card-upload');

      const chunk = new Uint8Array(256 * 1024); // 256KB chunks
      let totalBytes = 0;
      const durationMs = 5000;
      const startTime = performance.now();

      while (performance.now() - startTime < durationMs) {
        const t0 = performance.now();
        await fetch('/api/speedtest/upload?_=' + Date.now(), {
          method: 'POST',
          body: chunk
        });
        totalBytes += chunk.length;
        const elapsedSec = (performance.now() - startTime) / 1000;
        if (elapsedSec > 0.2) {
          const curMbps = ((totalBytes * 8) / elapsedSec) / 1000000;
          document.getElementById('liveSpeed').innerText = curMbps.toFixed(1);
          drawGauge(curMbps, 100);
        }
      }

      const totalSec = (performance.now() - startTime) / 1000;
      const finalMbps = ((totalBytes * 8) / totalSec) / 1000000;
      document.getElementById('val-upload').innerText = finalMbps.toFixed(1);
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
        document.getElementById('phaseText').innerText = 'Hata Oluştu';
      } finally {
        btn.disabled = false;
        setTimeout(() => {
          document.getElementById('liveSpeed').innerText = '0.0';
          drawGauge(0);
        }, 1500);
      }
    }

    // Auto-start on first load
    setTimeout(runSpeedtest, 600);
  </script>
</body>
</html>`
