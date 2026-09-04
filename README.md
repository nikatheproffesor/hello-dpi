# ⚡ Hello DPI

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey)](#-cross-platform-support)
[![Speed](https://img.shields.io/badge/Speed-100%25%20Line%20Rate-brightgreen)](#-why-not-vpn)

**Ultra-lightweight, zero-latency, cross-platform Deep Packet Inspection (DPI) circumvention proxy.**

[Türkçe Dokümantasyon](#-t%C3%BCrk%C3%A7e-dok%C3%BCmantasyon) • [English Documentation](#-english-documentation)

</div>

---

## 🇹🇷 Türkçe Dokümantasyon

### 🎯 Nedir ve Neden "Hello DPI"?

İnternet Servis Sağlayıcıları (İSS) ve sansür filtreleri, web sitelerine erişimi engellemek için **Derin Paket İncelemesi (DPI)** uygular. Bu filtreler, bağlantı kurulurken şifrelenmemiş olan **TLS ClientHello (SNI)** veya **HTTP Host** paketlerini yakalayarak engelleme yapar.

**Hello DPI**, bilgisayarınızda arka planda çalışan sıfır yük bindiren yerel bir vekil sunucudur (local proxy). Bağlantı anında SNI ve Host başlıklarını mikro TCP parçalarına (fragmentation) bölerek sansür kutularının bu paketleri okumasını engeller, ancak hedef sunucunun doğru okuyabilmesini sağlar.

### 🚀 Neden VPN Değil?

| Özellik | Geleneksel VPN | GoodbyeDPI | **Hello DPI** |
| :--- | :--- | :--- | :--- |
| **İnternet Hızı** | 🔻 %30-%70 yavaşlama | ⚡ %100 Hat Hızı | ⚡ **%100 Hat Hızı** |
| **Gecikme (Ping)** | 🔻 +50ms - +200ms artış | 🟢 0ms ek gecikme | 🟢 **0ms ek gecikme** |
| **Platform Desteği** | Mac, Win, Linux | ❌ Yalnızca Windows | 🍏 **macOS**, 🪟 **Windows**, 🐧 **Linux** |
| **Kernel / Sürücü** | TAP/TUN sürücüsü ister | WinDivert (Kernel) ister | 🛡️ **Sürücüsüz, Güvenli (User-space)** |
| **Kaynak Tüketimi**| Yüksek RAM ve CPU | Düşük | 🪶 **< 15 MB RAM, %0 CPU** |
| **DNS Güvenliği** | Uzak tünelden | Sistem DNS | 🔒 **Dahili DNS-over-HTTPS (DoH)** |

---

### 💻 Hızlı Başlangıç

#### 1. Kaynak Koddan Çalıştırma veya Derleme
Go yüklü ise tek komutla çalıştırabilirsiniz:
```bash
# Projeyi klonlayın
git clone https://github.com/hellodpi/hellodpi.git
cd hellodpi

# Doğrudan çalıştırın
go run ./cmd/hellodpi -system-proxy
```

#### 2. macOS Kurulumu (Arka Planda Otomatik Çalışma)
```bash
chmod +x scripts/install-macos.sh
./scripts/install-macos.sh
```
*Bu komut Hello DPI'ı `launchd` servisi olarak arka plana yerleştirir. Bilgisayarınız her açıldığında otomatik ve sessizce çalışır.*

#### 3. Linux Kurulumu (systemd)
```bash
chmod +x scripts/install-linux.sh
./scripts/install-linux.sh
```

#### 4. Windows Kurulumu
`scripts/install-windows.bat` dosyasına çift tıklayın. Otomatik olarak Başlangıç klasörünüze görünmez arka plan servisi olarak eklenir.

---

### ⚙️ Komut Satırı Seçenekleri

```bash
hellodpi [seçenekler]

Seçenekler:
  -addr string
        Proxy dinleme adresi (varsayılan: "127.0.0.1:8080")
  -mode string
        Parçalama tekniği: 'sni', 'first-byte', 'chunked' (varsayılan: "sni")
  -delay int
        Parçalar arası bekleme süresi ms cinsinden (varsayılan: 2)
  -doh
        DNS-over-HTTPS (DoH) ile güvenli alan adı çözümleme (varsayılan: true)
  -doh-server string
        DoH sunucu adresi (varsayılan: Cloudflare "https://1.1.1.1/dns-query")
  -system-proxy
        Açılışta işletim sistemi proxy ayarlarını otomatik açar, kapanışta geri alır
  -version
        Sürüm bilgisini gösterir
```

---

## 🇬🇧 English Documentation

### 🎯 What is Hello DPI?
Internet Service Providers (ISPs) and network censors inspect packets using **Deep Packet Inspection (DPI)** to block domain names by inspecting the plain-text **SNI (Server Name Indication)** field in the TLS ClientHello or plain HTTP `Host` headers.

**Hello DPI** is a lightweight, cross-platform local evasion proxy. It fragments the initial handshake packet into microscopic TCP segments before transmitting them. The DPI inspection boxes fail to inspect the fragmented payload, while the destination web server reassembles them effortlessly.

### 🛡️ How it Works

```
[ Browser / App ]
       │  (Local Traffic: 127.0.0.1:8080)
       ▼
[ Hello DPI Core ]
  ├── 1. Dual HTTP CONNECT & RFC 1928 SOCKS5 Listener
  ├── 2. DNS-over-HTTPS (DoH) Resolver (Bypasses ISP DNS Poisoning)
  ├── 3. TLS ClientHello & SNI Fragmentation Engine
  │      - Splits hostname exactly midway across TCP segment boundaries
  │      - Enforces TCP_NODELAY to avoid packet coalescence
  └── 4. High-Speed Bidirectional Zero-Copy Pipe (io.CopyBuffer)
         │  (Direct Connection at 100% Native ISP Fiber Speed)
         ▼
[ Target Server (port 443 / 80) ]
```

### 📦 Pre-built Releases
Pre-compiled standalone binaries for **macOS (Apple Silicon & Intel)**, **Windows (x64)**, and **Linux (x64 & ARM64)** are available on the [GitHub Releases](https://github.com/hellodpi/hellodpi/releases) tab. No installation or dependencies needed.

### 🤝 Contributing
Pull requests, issues, and feature suggestions are welcome!

### 📜 License
This project is licensed under the [MIT License](LICENSE).
