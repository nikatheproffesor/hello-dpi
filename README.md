<div align="center">

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo_white.png">
    <img src="assets/logo.png" width="120" alt="Hello DPI Logo" />
  </picture>

  # Hello DPI

  **İnternetinizi yavaşlatmayan, terminal gerektirmeyen, tek tıkla çalışan sansür aşma aracı.**
  
  *Ultra-lightweight, zero-latency, cross-platform DPI circumvention tool with a native menubar/tray app.*

  <br />

  [![License: MIT](https://img.shields.io/badge/Lisans-MIT-black.svg)](LICENSE)
  [![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Linux-black.svg)](#-nasıl-kullanılır)
  [![Speed](https://img.shields.io/badge/H%C4%B1z-%25100%20Hat%20H%C4%B1z%C4%B1-black.svg)](#-neden-vpn-değil)
  [![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)

  <br />

  [🇹🇷 Türkçe](#-türkçe-rehber) • [🇬🇧 English](#-english-guide) • [🚀 Hemen İndir](#-hemen-indir)

</div>

---

## 🇹🇷 Türkçe Rehber

### 🚀 Hemen İndir

İşletim sisteminize uygun olanı seçip tek tıkla kullanmaya başlayın:

| İşletim Sistemi | İndirme Linki | Kurulum |
| :--- | :--- | :--- |
| **🍏 macOS (DMG - Önerilen)** | [**HelloDPI-macOS.dmg**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | `.dmg` dosyasını açıp `Hello DPI.app` uygulamasını Applications klasörüne sürükleyin. |
| **🍏 macOS (ZIP)** | [**HelloDPI-macOS.zip**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | `.zip` dosyasını açın ve `Hello DPI.app` uygulamasını çalıştırın. |
| **🪟 Windows** | [**HelloDPI-Windows.exe**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | İndirin ve çift tıklayın (Siyah konsol açılmaz, sağ altta saatin yanına yerleşir). |
| **🐧 Linux** | [**hellodpi-linux-amd64**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | Terminalden tek komutla veya systemd servisi olarak çalıştırın. |

> [!TIP]
> **🍏 Mac Kullanıcıları İçin Önemli Not ("Hasar Görmüş" Uyarısı Alırsanız):**
> Apple, yıllık $99 geliştirici lisansı bulunmayan açık kaynaklı tüm bağımsız uygulamalara internetten indirildiğinde otomatik güvenlik karantinası koyar.
> 
> Çözümü çok basittir:
> 1. İndirdiğiniz klasördeki **`Başlat.command`** dosyasına çift tıklayın, **VEYA**
> 2. Terminale şu tek satırı yapıştırıp Enter'a basın:
>    ```bash
>    xattr -cr ~/Downloads/"Hello DPI.app"
>    ```
> *(Uygulama Applications klasöründeyse: `xattr -cr /Applications/"Hello DPI.app"`)*

---

### ❓ Hello DPI Nedir?

Türkiye'de Discord ve engellenen siteler, İnternet Servis Sağlayıcılarının (Türk Telekom, TurkNet, Superonline vb.) uyguladığı **DPI (Derin Paket İncelemesi)** filtreleri yüzünden açılamaz.

**Hello DPI**, internetinizi başka ülkelere yönlendirip yavaşlatan hantal VPN programlarının aksine:
- ⚡ **İnternet Hızınızı Asla Düşürmez:** %100 kendi fiber veya hat hızınızda çalışır.
- 🎮 **Oyunlarda Pinginizi Artırmaz:** Bağlantı doğrudan hedefe gider; gecikme eklenmez (0ms).
- 🛡️ **Arka Planda Sessizce Korur:** Bilgisayarınızı yormaz, %0 işlemci (CPU) ve sadece 15 MB RAM kullanır.
- 🖱️ **Terminal Açtırmaz:** Ekranın köşesindeki simgesinden (👋) tek tıkla açılıp kapanabilir.

---

### 💡 2 Adımda Kullanım

1. Size uygun dosyayı yukarıdaki tablodan indirin.
2. Çift tıklayarak çalıştırın:
   - **Mac'te:** Sağ üst köşedeki menü çubuğunda el sallama simgemiz (👋) belirir.
   - **Windows'ta:** Sağ altta saatin yanındaki sistem tepsisine simgemiz yerleşir.
3. **Bitti!** Artık Discord'u ve tüm siteleri dilediğiniz gibi açabilirsiniz. Korumayı geçici olarak durdurmak isterseniz simgeye tıklayıp *"Korumayı Duraklat"* demeniz yeterlidir.

---

### 📊 Hello DPI vs VPN vs GoodbyeDPI

| Özellik | Geleneksel VPN | GoodbyeDPI | ⚡ **Hello DPI** |
| :--- | :--- | :--- | :--- |
| **Kullanım Kolaylığı** | Hesap & Abonelik ister | Karmaşık `.cmd` dosyaları | 🖱️ **Tek tıkla çalışır (Menü Çubuğu)** |
| **İnternet Hızı** | 🔻 %30-%70 yavaşlama | ⚡ %100 Hat Hızı | ⚡ **%100 Tam Hat Hızı** |
| **Oyun Pingi** | 🔻 +50ms ile +200ms gecikme | 🟢 0ms ek gecikme | 🟢 **0ms (Sıfır Ping Etkisi)** |
| **Platform** | Farklı uygulamalar | ❌ Yalnızca Windows | 🍏 **Mac**, 🪟 **Windows**, 🐧 **Linux** |
| **Discord Erişimi** | Yavaş / Bazen engelli | Türkiye için karmaşık ayar | 🛡️ **RFC TLS Record Splitting ile Hazır** |
| **Bilgisayara Yükü** | Ağır arka plan servisi | Düşük | 🪶 **< 15 MB RAM, %0 Boşta CPU** |

---

<details>
<summary><b>🛠️ Nasıl Çalışır? (Meraklısına Teknik Detaylar)</b></summary>

<br>

Klasik sansür filtreleri, HTTPS trafiği şifrelenmeden hemen önceki ilk paket olan **TLS ClientHello (SNI)** alanını okuyarak engelleme uygular. 

Hello DPI, **RFC 5246 ve RFC 8446** standartlarına tam uyumlu olarak bu ilk paketi iki geçerli TLS kaydına (TLS Record) böler:
1. **1. Kayıt:** Sadece el sıkışma başlığını taşır; içinde hedef web sitesinin ismi (SNI) yoktur. İSS filtreleri paketi temiz sanarak geçirir.
2. **2. Kayıt:** Kalan veriyi taşır. Filtreler yeni bir el sıkışma görmediği için atlar.
3. **Hedef Sunucu (Discord / Cloudflare):** Standart gereği iki kaydı anında birleştirip güvenli HTTPS tünelini açar.

Ayrıca İSS'lerin DNS zehirlemesini engellemek için dahili **Cloudflare DNS-over-HTTPS (DoH)** motoru barındırır.
</details>

---

## 🇬🇧 English Guide

### 🎯 What is Hello DPI?
Hello DPI is a zero-latency, cross-platform Deep Packet Inspection (DPI) circumvention proxy with an ultra-minimalist menu bar / system tray interface. It lets users bypass ISP-level domain censorship without the speed penalty or high ping associated with traditional VPNs.

### 🚀 Key Features
- **No VPN Speed Degradation:** Connects directly to the destination at 100% native line speed.
- **Zero Configuration GUI:** Lives in the macOS Menu Bar and Windows System Tray with 1-click controls.
- **State-of-the-Art Evasion:** Implements RFC-compliant TLS Record Splitting and built-in DNS-over-HTTPS.
- **Ultra Lightweight:** Consumes under 15 MB RAM and 0% idle CPU.

---

### 📦 Quick Start (Developers & CLI)

```bash
git clone https://github.com/emreaytekxn/hello-dpi.git
cd hello-dpi

# Build the Menu Bar / Tray GUI:
go build -o "Hello DPI.app/Contents/MacOS/Hello DPI" ./cmd/hellodpi-tray

# Run CLI standalone:
go run ./cmd/hellodpi -system-proxy
```

---

### 📜 License
This project is open source and available under the [MIT License](LICENSE).
