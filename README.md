<div align="center">

<img src="assets/logo.png" width="160" alt="Hello DPI Logo" style="border-radius: 24px; box-shadow: 0 10px 30px rgba(0, 200, 255, 0.2);" />

# ⚡ Hello DPI

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey)](#-cross-platform-support)
[![Speed](https://img.shields.io/badge/Speed-100%25%20Line%20Rate-brightgreen)](#-why-not-vpn)

**Terminal bilgisi gerektirmeyen, sıfır gecikmeli, ultra hafif çapraz platform DPI & Sansür Aşma Aracı.**

[Türkçe Dokümantasyon](#-t%C3%BCrk%C3%A7e-dok%C3%BCmantasyon) • [English Documentation](#-english-documentation)

</div>

---

## 🇹🇷 Türkçe Dokümantasyon

### 🎯 Nedir ve Neden "Hello DPI"?

İnternet Servis Sağlayıcıları (İSS), Discord ve benzeri siteleri engellemek için **Derin Paket İncelemesi (DPI)** uygular. 

**Hello DPI**, bilgisayarınızda sessizce arka planda çalışan, **VPN gibi internetinizi yavaşlatmayan**, ping değerinizi artırmayan ve terminal açmayı bilmeyen herkesin **tek bir tıklamayla** kullanabileceği açık kaynaklı bir araçtır.

---

### 🖱️ Kolay Kullanım (Terminal Gerekmez!)

Sıradan kullanıcılar için hiçbir komut satırı veya ayar gerekmez:

#### 🍏 macOS (Mac Kullanıcıları)
1. [Releases](https://github.com/hellodpi/hellodpi/releases) sayfasından **`HelloDPI-macOS.zip`** dosyasını indirin.
2. Çift tıklayarak açın ve içindeki **`Hello DPI.app`** uygulamasını çalıştırın.
3. Sağ üst menü çubuğunuzda kalkan simgesi 🛡️ belirecektir.
4. Discord'u ve tüm siteleri dilediğiniz gibi sansürsüz kullanabilirsiniz!

#### 🪟 Windows Kullanıcıları
1. [Releases](https://github.com/hellodpi/hellodpi/releases) sayfasından **`HelloDPI-Windows.exe`** dosyasını indirin.
2. Çift tıklayın. Siyah konsol penceresi açılmaz; doğrudan sağ alt sistem tepsisine (saatin yanına) yerleşir.
3. Simgeye sağ tıklayarak korumayı duraklatabilir, bağlantı testi yapabilir veya çıkabilirsiniz.

---

### 🚀 Neden VPN Değil?

| Özellik | Geleneksel VPN | GoodbyeDPI | **Hello DPI** |
| :--- | :--- | :--- | :--- |
| **Kullanım Kolaylığı** | Hesap, abonelik ister | Karmaşık CMD dosyaları | 🖱️ **Tek tıkla aç/kapa (Menü Çubuğu)** |
| **İnternet Hızı** | 🔻 %30-%70 yavaşlama | ⚡ %100 Hat Hızı | ⚡ **%100 Hat Hızı** |
| **Gecikme (Ping)** | 🔻 +50ms - +200ms | 🟢 0ms ek gecikme | 🟢 **0ms ek gecikme** |
| **Platform Desteği** | Çeşitli | ❌ Yalnızca Windows | 🍏 **macOS**, 🪟 **Windows**, 🐧 **Linux** |
| **Discord Erişimi** | Yavaş / Bazen engelli | Türkiye için özel ayar ister | 🛡️ **RFC TLS Record Splitting ile Hazır** |
| **Kaynak Tüketimi**| Yüksek RAM ve CPU | Düşük | 🪶 **< 15 MB RAM, %0 Boşta CPU** |

---

### 💻 Geliştiriciler & İleri Düzey Kullanıcılar İçin (CLI)

```bash
# Kaynak koddan derleme ve çalıştırma
git clone https://github.com/hellodpi/hellodpi.git
cd hellodpi

# Menü çubuğu uygulamasını derleme:
go build -o "Hello DPI.app/Contents/MacOS/Hello DPI" ./cmd/hellodpi-tray

# Komut satırı (CLI) sürümü:
go run ./cmd/hellodpi -system-proxy
```

---

## 🇬🇧 English Documentation

### 🎯 What is Hello DPI?
Hello DPI is an ultra-lightweight, cross-platform DPI circumvention tool designed with a friendly **Menu Bar / System Tray** interface. It allows non-technical users to bypass internet censorship without touching the terminal or degrading internet speed.

### 📦 Download & Run
Pre-built standalone apps are available on the [Releases](https://github.com/hellodpi/hellodpi/releases) page:
- **macOS:** Download `HelloDPI-macOS.zip`, unzip, and run `Hello DPI.app`.
- **Windows:** Download and run `HelloDPI-Windows.exe` (silent tray application, no console window).

### 📜 License
Licensed under the [MIT License](LICENSE).
