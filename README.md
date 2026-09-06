<div align="center">

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo_white.png">
    <img src="assets/logo.png" width="120" alt="Hello DPI Logo" />
  </picture>

  # Hello DPI

  **Terminal gerektirmeyen, internet hızınızı ve oyun pinginizi düşürmeyen, tek tıkla çalışan yeni nesil sansür aşma aracı.**

  *Next-generation, zero-latency, cross-platform DPI evasion tool with native system tray, mobile Quick Settings & auto-updater.*

  <br />

  [![Release](https://img.shields.io/github/v/release/nikatheproffesor/hello-dpi?color=black&logo=github&label=S%C3%BCr%C3%BCm%20v5.0.0)](https://github.com/nikatheproffesor/hello-dpi/releases/latest)
  [![VirusTotal](https://img.shields.io/badge/VirusTotal-0%2F72%20Temiz-brightgreen?logo=virustotal)](https://www.virustotal.com/gui/file/30037fd5b5d1a32bf15c5bf4c861422b2f0e07e661a9858de39e104e276ad732)
  [![Apple Signed](https://img.shields.io/badge/Apple%20Signed-Developer%20ID-black?logo=apple)](https://github.com/nikatheproffesor/hello-dpi/releases/latest)
  [![Ping](https://img.shields.io/badge/Ping-0%20ms%20Ek%20Gecikme-black.svg)](#-neden-vpn-değil)
  [![Hız](https://img.shields.io/badge/H%C4%B1z-%25100%20Hat%20H%C4%B1z%C4%B1-black.svg)](#-neden-vpn-değil)
  [![Lisans](https://img.shields.io/badge/Lisans-MIT-black.svg)](LICENSE)

  <br />

  [📥 Hemen İndir](#-hemen-indir-v500) • [⚡ Neden VPN Değil?](#-neden-vpn-değil) • [✨ Özellikler](#-öne-çıkan-özellikler-v50) • [🩺 Ağ Doktoru](#-ağ-doktoru-ve-canlı-ping-monitörü) • [📊 Karşılaştırma](#-karşılaştırma-tablosu) • [❓ SSS](#-sıkça-sorulan-sorular) • [🇬🇧 English Guide](#-english-guide)

</div>

---

## 📥 Hemen İndir (v5.0.0)

Hiçbir terminal, kod veya karmaşık ayar gerektirmez. İşletim sisteminize uygun dosyayı indirip doğrudan çalıştırabilirsiniz:

| Platform | İndirme Bağlantısı | Açıklama |
| :--- | :--- | :--- |
| **🪟 Windows** | [**HelloDPI-Setup.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-Setup.exe) | **1-Tık Kurucu (Önerilen).** Masaüstü kısayolu oluşturur, sistem tepsisine yerleşir ve arka planda sessizce çalışır. |
| **🍏 macOS** | [**HelloDPI-macOS.dmg**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-macOS.dmg) | **Resmi Apple İmzalı DMG.** Applications klasörüne sürükleyin, menü çubuğundan tek tıkla kontrol edin. |
| **🤖 Android** | [**HelloDPI-Android.apk**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-Android.apk) | **1-Dokunuş Kurulum.** Bildirim paneli Hızlı Ayarlar kutucuğu (Quick Settings Tile), sessiz arka plan servisi. |
| **🐧 Linux** | [**hellodpi-linux-amd64**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/hellodpi-linux-amd64) | **64-bit Bağımsız İkili.** GNOME / KDE masaüstü tepsisi veya bağımsız komut satırı servisi. |

> ℹ️ *iOS sürümü şu anda geliştirme aşamasındadır (TestFlight sürümü yakında kullanıma sunulacaktır).*

<details>
<summary><b>📦 Gelişmiş & Alternatif Paketler</b></summary>

- **macOS ZIP Paketi:** [HelloDPI-macOS.zip](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-macOS.zip) (DMG açmak istemeyenler için doğrudan uygulama arşivi)
- **Android ARM64 CLI:** [hellodpi-android-arm64](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/hellodpi-android-arm64) (Termux / kök kullanıcıları için bağımsız çekirdek)
- **Tüm Dosyalar ve Değişiklik Günlüğü:** [GitHub Releases](https://github.com/nikatheproffesor/hello-dpi/releases)

</details>

---

## ⚡ Neden VPN Değil?

Geleneksel VPN'ler tüm internet trafiğinizi yurt dışındaki (Almanya, Hollanda vb.) uzak sunuculara yönlendirir. Bu durum pinginizi 100-200 ms artırır, hat hızınızı yarı yarıya düşürür ve anti-cheat sistemlerinde (Riot Vanguard, BattlEye) hesap banlanma riski oluşturur.

**Hello DPI bir VPN sunucusu değildir.** Trafiğinizi doğrudan kendi ev internetinizden hedefe iletir; yalnızca bağlantı kurulurken hedefe giden ilk paketi (TLS ClientHello) akıllıca parçalayarak servis sağlayıcınızın sansür filtrelerini (DPI) etkisiz hale getirir.

```
[ GELENEKSEL VPN (Yavaş, Gecikmeli ve Güvensiz) ]
Siz ───> [ Hollanda / Almanya VPN Sunucusu ] ───> Discord / Roblox / Web
         🔻 %50 - %70 Hız Kaybı
         🔻 +80ms - +200ms Ek Ping Gecikmesi
         🔻 Tüm trafiğiniz ve şifreleriniz yabancı sunucudan geçer
         🔻 Riot Vanguard / Anti-Cheat şüpheli yabancı IP nedeniyle banlayabilir

[ HELLO DPI v5.0 (Doğrudan, Şeffaf ve Işık Hızında) ]
Siz ═════════════════════════════════════════════> Discord / Roblox / Web
     ⚡ 0 ms Ek Ping (Doğal hat gecikmeniz neyse odur)
     ⚡ %100 Tam Fiber Hat Hızı (1000 Mbps ise 1000 Mbps)
     ⚡ Kendi Yerel IP Adresiniz (Anti-cheat dostu, sıfır ban riski)
     ⚡ Sıfır Terminal (Tek tıkla arka planda sessizce çalışır)
```

---

## ✨ Öne Çıkan Özellikler (v5.0)

- 🖱️ **Sıfır Terminal & 100% Grafiksel Arayüz:** Siyah komut satırı ekranları veya karmaşık parametreler yoktur. Saatin yanındaki sistem tepsisinde (Windows) veya menü çubuğunda (macOS) minimalist bir simge olarak yaşar.
- ⚡ **0 ms Ek Ping & %100 Hat Hızı:** Trafik yabancı bir sunucuya gitmez. Discord ses kanallarında veya oyunlarda (Valorant, CS2, LoL, Roblox) gecikme yaşanmaz.
- 🛡️ **Gelişmiş DPI Atlama (Decoy & Out-of-Order):** Sandvine, Procera ve Huawei kurumsal DPI donanımlarını sahte paket enjeksiyonu ve RFC 5246/8446 TLS kayıt parçalama yöntemiyle şaşırtarak sansürü tamamen etkisiz kılar.
- 🩺 **Ağ Doktoru & Canlı Ping Monitörü:** Discord Voice (Frankfurt, Rotterdam), Roblox ve Cloudflare sunucularına doğal hat pinginizi ve servis erişim durumunuzu tek tıkla canlı olarak test edin.
- 🚀 **Dahili 60 FPS Hız Testi:** Gerçek Cloudflare Edge CDN üzerinden çalışan, ibreli hız testi paneliyle anlık indirme, yükleme ve jitter değerlerinizi ölçün.
- 📱 **Android Mobil Desteği:** Root gerektirmeyen yerel motoru, bildirim paneli Hızlı Ayarlar kutucuğu (Quick Settings Tile) ve açılışta otomatik başlatma özelliğiyle akıllı telefonunuzda da tam koruma sağlar.
- 🔄 **Tek Tıkla Uygulama İçi Güncelleme:** Yeni bir sürüm çıktığında sistem tepsisinden tek tıkla arka planda güncellenir; yeniden dosya indirip kurmanıza gerek kalmaz.
- 🏢 **GSB WiFi (KYK Yurt İnterneti) Tam Uyumluluğu:** Captive portal bypass mimarisi sayesinde yurt internetinde giriş sayfası donmadan açılır, giriş yapıldıktan sonra tüm kısıtlamalar kendiliğinden kalkar.
- 🔒 **Sıfır Risk & %100 Güvenli:** Bilgisayarınıza kök sertifika (MITM CA) yüklemez. Şifreli HTTPS trafiğinizi okuyamaz ve kaydedemez. Dünyanın önde gelen 70+ antivirüs motorunda **0/72 Temiz** olarak doğrulanmıştır.

---

## 🩺 Ağ Doktoru ve Canlı Ping Monitörü

Sistem tepsisinden veya menü çubuğundan **Ağ Doktoru** seçeneğine tıkladığınızda açılan kontrol panelinde:

- **Canlı Gecikme Ölçümü:** Discord Voice Avrupa sunucuları (Frankfurt, Rotterdam), Roblox ve Cloudflare üzerindeki ping değerlerinizi milisaniye cinsinden canlı izleyin.
- **Tek Tıkla Ağ Onarımı:** Zehirlenmiş DNS önbelleğini temizler, proxy yönlendirmelerini doğrular ve olası bağlantı takılmalarını otomatik çözer.
- **0 ms VPN Avantajı:** VPN servislerinin aksine doğal internet hızınızın ve pinginizin korunduğunu canlı grafiklerle doğrulayın.

---

## 🚀 Hızlı Başlangıç

### 🪟 Windows
1. [**HelloDPI-Setup.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-Setup.exe) dosyasını indirin ve çift tıklayın.
2. Kurulum tamamlandığında uygulama arka planda sessizce başlar ve saatin yanındaki sistem tepsisine yerleşir.
3. Discord, Roblox ve sansürlü web sitelerine doğrudan erişebilirsiniz.

<details>
<summary><b>Windows SmartScreen ("Windows kişisel bilgisayarınızı korudu") uyarısı çıkarsa:</b></summary>

Açık kaynaklı yeni bağımsız yazılımlarda Windows standart bir güvenlik uyarısı gösterebilir:
1. Mavi uyarı kutusundaki **"Ek Bilgi" (More info)** bağlantısına tıklayın.
2. Beliren **"Yine de Çalıştır" (Run anyway)** butonuna tıklayın.
Windows bu tercihi hafızaya alır ve sonraki açılışlarda bir daha sormaz.
</details>

---

### 🍏 macOS
1. [**HelloDPI-macOS.dmg**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-macOS.dmg) dosyasını açın.
2. `Hello DPI` simgesini yanındaki `Applications` klasörüne sürükleyin.
3. Uygulamayı çalıştırın; sağ üst menü çubuğunda simgemiz belirecektir.

<details>
<summary><b>Mac'te "Uygulama Hasar Görmüş" veya "Doğrulanamadı" uyarısı çıkarsa:</b></summary>

Apple Gatekeeper'ın açık kaynaklı bağımsız uygulamalara koyduğu standart denetimdir:
1. Ekrana gelen uyarıda **"Vazgeç"** deyin.
2. **Sistem Ayarları (System Settings) > Gizlilik ve Güvenlik (Privacy & Security)** sekmesini açın.
3. Sayfanın en altındaki *"Hello DPI engellendi"* uyarısının yanındaki **"Yine de Aç" (Open Anyway)** butonuna tıklayın.
</details>

---

### 🤖 Android
1. [**HelloDPI-Android.apk**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-Android.apk) dosyasını telefonunuza indirin ve kurun.
2. Uygulamayı açıp **"Başlat"** butonuna dokunun.
3. İsterseniz telefonunuzun üst bildirim panelini aşağı kaydırıp **Hızlı Ayarlar (Tile)** arasına Hello DPI'ı ekleyebilir; uygulamayı dahi açmadan tek dokunuşla kontrol edebilirsiniz.

---

## 📊 Karşılaştırma Tablosu

| Özellik | Geleneksel VPN | GoodbyeDPI | Zapret | ⚡ **Hello DPI v5.0** |
| :--- | :--- | :--- | :--- | :--- |
| **Kullanım Kolaylığı** | Hesap / Abonelik | `.cmd` komut dosyaları | Terminal & root | 🖱️ **Tek tıkla grafiksel arayüz (100% GUI)** |
| **Terminal / Kod Gereksinimi** | Yok | Var | Var | 🟢 **SIFIR TERMİNAL** |
| **Otomatik Güncelleme** | Var | ❌ Manuel | ❌ Manuel | ⚡ **Tek tıkla uygulama içinden** |
| **İnternet Hızı** | 🔻 %50 - %70 Düşüş | ⚡ %100 Hat Hızı | ⚡ %100 Hat Hızı | ⚡ **%100 Tam Hat Hızı (Fiber)** |
| **Oyun Pingi (Gecikme)** | 🔻 +80ms - +200ms | 🟢 0 ms ek ping | 🟢 0 ms ek ping | 🟢 **0 ms (Sıfır Ek Gecikme)** |
| **Anti-Cheat Uyumluluğu** | ⚠️ Ban riski var | 🟢 Güvenli | 🟢 Güvenli | 🟢 **%100 Güvenli (Kendi IP'niz)** |
| **Platform Desteği** | Çeşitli | Yalnızca Windows | Linux ağırlıklı | 🪟 **Windows**, 🍏 **macOS**, 🤖 **Android**, 🐧 **Linux** |
| **Mobil Uygulama** | Ağır VPN istemcisi | ❌ Yok | ❌ Yok | 🤖 **Android APK (Hızlı Ayarlar Kutucuğu)** |
| **Ağ Doktoru & Canlı Ping** | Yok | Yok | Yok | 🩺 **Dahili Canlı Monitör** |
| **Dahili Hız Testi** | Reklamlı / Harici | Yok | Yok | ⚡ **60 FPS Dahili Hız Testi** |
| **KYK / GSB WiFi Desteği** | Çoğu bloklu | ❌ DNS kilitlenir | ❌ Manuel ayar | 🛡️ **Otomatik Captive Portal Bypass** |
| **Sistem Kaynak Tüketimi** | Yüksek CPU & RAM | Düşük | Düşük | 🪶 **< 15 MB RAM, %0 CPU** |

---

## 🛡️ Güvenlik ve Doğruluk Güvencesi

Hello DPI, açık kaynaklı ve şeffaf bir projedir:
- Bilgisayarınıza veya telefonunuza **kök güvenlik sertifikası (MITM CA) yüklemez**.
- HTTPS trafiğinizin şifresini çözemez, özel mesajlarınızı veya bankacılık verilerinizi göremez.
- Tüm ikili dosyalar her sürümde VirusTotal üzerinde 70+ antivirüs motoruyla taranır:

| Platform / Dosya | SHA-256 Özeti | VirusTotal Raporu |
| :--- | :--- | :--- |
| **HelloDPI-Setup.exe** | `5f5ad4bedd4abe5f230f307d0f55f37a7a9523e73ff31e1ffbdd219c576505d6` | [**0/72 Temiz**](https://www.virustotal.com/gui/file/30037fd5b5d1a32bf15c5bf4c861422b2f0e07e661a9858de39e104e276ad732) |
| **HelloDPI-macOS.dmg** | `186e61b769da7c10db0d60bb9d0e89e579d25afae7d9f69d03362e5a9579ae39` | [**0/65 Temiz**](https://www.virustotal.com/gui/file/b7b159fb9568f266fae2f1f2c9a75518417658f8b40c415db7bf8f38795ec652) |
| **HelloDPI-Android.apk** | `65ce747654581a17ac5800476ad8740e9a464bc02bbd627f42be31cbc1283038` | [**0/65 Temiz**](https://www.virustotal.com/gui/file/844d5272908215776cb5d339815bdaedf7e0ad41698ab2294706c04e0aa673fd) |
| **hellodpi-linux-amd64** | `a0625d3adbbc7378eae2962d3a67716f4eef583a9cfa3782c8332feef3e12640` | [**0/65 Temiz**](https://www.virustotal.com/gui/file/00b22ca3b8bb24b2fc1556ecac8811371039ae4cc2483a573dcd6a898957bec1) |

---

## ❓ Sıkça Sorulan Sorular

<details>
<summary><b>1. Valorant, CS2 veya LoL oynarken ban yer miyim?</b></summary>
<br>

**Kesinlikle hayır.** VPN servisleri IP adresinizi yabancı ülkelere taşıdığı için Riot Vanguard veya BattlEye gibi hile koruma sistemleri bunu şüpheli konum olarak algılayıp hesabınızı geçici olarak durdurabilir. Hello DPI ise **IP adresinizi asla değiştirmez.** Siz yine kendi ev internetinizin Türk Telekom, Superonline veya TurkNet IP'si ile doğrudan oyuna bağlanırsınız. Pinginiz milisaniye dahi artmaz.
</details>

<details>
<summary><b>2. Watch Together veya video senkronizasyon odaları çalışıyor mu?</b></summary>
<br>

**Evet.** Hello DPI'ın `bufferedConn` mimarisi sayesinde Watch Together (w2g.tv), Kosmi ve benzeri tüm WebSockets (`wss://`) ve HTTP/2 akışları tek bir baytı kaybolmadan tam hat hızında çalışır.
</details>

<details>
<summary><b>3. KYK (GSB WiFi) yurt internetinde çalışır mı?</b></summary>
<br>

**Evet.** KYK yurtlarında internete çıkabilmek için önce `wifi.gsb.gov.tr` portalından giriş yapılması gerekir. Hello DPI bu adresleri otomatik tanıyarak doğrudan yerel ağa yönlendirir. Giriş sayfanız takılmadan açılır; giriş yapıldıktan sonra ise tüm sansürsüz internet koruması kendiliğinden devreye girer.
</details>

<details>
<summary><b>4. Discord masaüstü uygulamasında ses kanalları açılmıyor, ne yapmalıyım?</b></summary>
<br>

Discord önceden arka planda açıksa eski engelli oturumu önbellekte tutmuş olabilir:
1. Discord uygulamasını tamamen kapatın (Mac'te `Cmd + Q`, Windows'ta Görev Yöneticisi veya sistem tepsisinden çıkış).
2. Hello DPI'ın çalıştığından emin olun.
3. Discord'u tekrar açın; ses kanallarına gecikmesiz bağlandığınızı göreceksiniz.
</details>

<details>
<summary><b>5. Türkiye'de bu programı kullanmak yasal mıdır?</b></summary>
<br>

**Evet, kişisel kullanım tamamen yasaldır.** 5651 Sayılı Kanun kapsamında vatandaşların DNS, VPN veya DPI manipülasyonu araçları kullanarak internete erişmesi suç teşkil etmez. Hukuken suç olan erişim yöntemi değil; internet üzerinde işlenebilecek yasa dışı eylemlerdir. Günlük internet, oyun ve Discord iletişiminizde hiçbir yasal sakınca bulunmamaktadır.
</details>

<details>
<summary><b>🛠️ Meraklısına Teknik Mimari (RFC 5246/8446)</b></summary>
<br>

DPI donanımları (Sandvine, Huawei vb.) servis sağlayıcı omurgasında paketleri inceler. Türkiye'deki İSS'ler çoğunlukla **stateful TCP reassembly** uygular. Hello DPI, **RFC 5246 (TLS 1.2)** ve **RFC 8446 (TLS 1.3)** standartlarının şu açık protokol kuralını uygular:
> *"Handshake messages MAY be coalesced into a single TLSPlaintext record, or divided among several records."*

1. **TLS Record Layer Splitting:** Gelen `ClientHello` paketi iki geçerli bağımsız TLS kaydına ayrıştırılır:
   - **1. Kayıt:** Yalnızca el sıkışma başlığını (5 bayt) taşır; içinde alan adı (SNI) yoktur. Sansür donanımı bu paketi zararsız bularak geçirir.
   - **2. Kayıt:** Kalan el sıkışma verisini taşır. Filtreler yeni bir el sıkışma başlangıcı görmediği için paketi denetlemeden atlar.
   - Hedef sunucu (Cloudflare, Discord vb.) iki kaydı RFC standardına göre hafızada birleştirerek güvenli şifreli oturumu kurar.
2. **Decoy & Out-of-Order:** Sandvine ve gelişmiş kurumsal filtreleri yanıltmak için sahte paket dizilimleri ve sıra dışı TCP akışı kullanılır.
3. **Multi-Tier Resilient DNS:** Cloudflare DoH ➔ Google DoH ➔ Quad9 DoH ➔ Yerel Sistem DNS yedekleme zinciri ile DNS sorguları asla yanıtsız kalmaz.
</details>

---

## 💻 Geliştiriciler İçin (Kaynak Koddan Derleme)

```bash
# 1. Depoyu klonlayın
git clone https://github.com/nikatheproffesor/hello-dpi.git
cd hello-dpi

# 2. Testleri çalıştırın
go test -v ./...

# 3. macOS Menü Çubuğu Uygulamasını derleyin:
./scripts/build-macos-app.sh

# 4. Windows Sistem Tepsisi Uygulamasını derleyin:
go build -ldflags="-H=windowsgui -s -w" -o "bin/HelloDPI-Windows.exe" ./cmd/hellodpi-tray

# 5. Bağımsız CLI motorunu çalıştırın:
go run ./cmd/hellodpi -system-proxy
```

---

## 🇬🇧 English Guide

### 🎯 Overview

Hello DPI is an ultra-lightweight, zero-latency Deep Packet Inspection (DPI) circumvention proxy with a native Menu Bar (macOS), System Tray (Windows), and Android Quick Settings interface. It bypasses ISP-level domain censorship without routing your traffic through remote VPN servers, granting you **100% native fiber line speed and 0 ms ping penalty**.

---

### 📥 1-Click Downloads (v5.0.0)

| Platform | Download Link | Notes |
| :--- | :--- | :--- |
| **🪟 Windows** | [**HelloDPI-Setup.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-Setup.exe) | 1-Click Setup, desktop shortcut, runs in system tray. |
| **🍏 macOS** | [**HelloDPI-macOS.dmg**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-macOS.dmg) | Official Apple Developer ID signed DMG for Menu Bar. |
| **🤖 Android** | [**HelloDPI-Android.apk**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/HelloDPI-Android.apk) | 1-Tap APK installation with Quick Settings Tile. |
| **🐧 Linux** | [**hellodpi-linux-amd64**](https://github.com/nikatheproffesor/hello-dpi/releases/download/v5.0.0/hellodpi-linux-amd64) | 64-bit standalone executable or systemd service. |

> ℹ️ *iOS version is currently in development (TestFlight coming soon).*

---

### 🚀 Key Features in v5.0

- **0 ms Extra Ping & 100% Native Speed:** No intermediary VPN servers. All packets flow directly from your ISP connection.
- **Zero Terminal:** 100% graphical interface sitting silently in your system tray or menu bar.
- **Decoy & RFC 5246/8446 Splitting:** Bypasses stateful DPI appliances (Sandvine, Huawei) by splitting the initial TLS ClientHello record.
- **Network Doctor & Live Ping:** Real-time ping monitor for Discord Voice (Frankfurt, Rotterdam), Roblox, and Cloudflare.
- **Built-in 60 FPS Speedometer:** Accurate HTML5 latency, download, upload, and jitter measurement.
- **Android Support:** Silent foreground service with Quick Settings Tile and auto-start on boot.
- **1-Click Auto-Update:** Seamless in-app binary updates directly from GitHub releases.
- **Anti-Cheat Safe:** Your public IP remains unchanged; zero risk of bans in Valorant, CS2, or League of Legends.

---

### ⚠️ Yasal Uyarı / Disclaimer

Bu yazılım yalnızca eğitim, ağ protokolleri araştırması (RFC 5246/8446) ve kişisel gizlilik testi amacıyla geliştirilmiştir. Kullanıcılar, bu yazılımı kullanarak gerçekleştirdikleri tüm eylemlerden ve tabi oldukları yerel mevzuata uyumdan bizzat sorumludur.

*This software is developed strictly for educational purposes, network protocol research (RFC 5246/8446), and personal privacy testing. Users are solely responsible for compliance with their local regulations.*

---

### 📜 Lisans / License

Bu proje [MIT Lisansı](LICENSE) altında açık kaynak olarak sunulmaktadır.  
*This project is open-source under the [MIT License](LICENSE).*
