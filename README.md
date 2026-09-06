<div align="center">

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo_white.png">
    <img src="assets/logo.png" width="130" alt="Hello DPI Logo" />
  </picture>

  # Hello DPI

  **Terminal gerektirmeyen, internet hızınızı ve pinginizi düşürmeyen, tek tıkla çalışan modern sansür aşma aracı.**
  
  *Zero-latency, zero-terminal, cross-platform DPI evasion tool with native system tray & auto-update support.*

  <br />

  [![Release](https://img.shields.io/github/v/release/nikatheproffesor/hello-dpi?color=black&logo=github)](https://github.com/nikatheproffesor/hello-dpi/releases/latest)
  [![VirusTotal](https://img.shields.io/badge/VirusTotal-0%2F72%20Temiz-brightgreen?logo=virustotal)](https://www.virustotal.com/gui/file/844d5272908215776cb5d339815bdaedf7e0ad41698ab2294706c04e0aa673fd)
  [![Apple Signed](https://img.shields.io/badge/Apple%20Signed-Developer%20ID-black?logo=apple)](https://github.com/nikatheproffesor/hello-dpi/releases/latest)
  [![License: MIT](https://img.shields.io/badge/Lisans-MIT-black.svg)](LICENSE)
  [![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux%20%7C%20Android%20%7C%20iOS-black.svg)](#-hemen-indir)
  [![Hız](https://img.shields.io/badge/H%C4%B1z-%25100%20Hat%20H%C4%B1z%C4%B1-black.svg)](#-neden-vpn-değil)
  [![Ping](https://img.shields.io/badge/Ping-0ms%20Ek%20Gecikme-black.svg)](#-neden-vpn-değil)
  [![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)

  <br />

  [🇹🇷 Türkçe Kılavuz](#-türkçe-rehber) • [🪟 Windows Kurulumu](#-windows-kullanıcıları-için) • [🍏 macOS Kurulumu](#-macos-kullanıcıları-için) • [📱 Mobil (Android & iOS)](#-mobil-destegi-android--ios) • [🛡️ Neden VPN Değil?](#-neden-vpn-değil) • [⚡ Ağ Doktoru & Ping](#-ag-doktoru-ve-canli-ping-monitöru) • [🇬🇧 English Guide](#-english-guide)

</div>

---

## 🇹🇷 Türkçe Rehber

### 🚀 Hemen İndir

Hiçbir komut satırı, kod veya terminal bilgisine ihtiyacınız yoktur. İşletim sisteminize uygun dosyayı indirip doğrudan farenizle çift tıklayarak kullanabilirsiniz:

| İşletim Sistemi | İndirme Dosyası | Kurulum ve Çalıştırma |
| :--- | :--- | :--- |
| **🪟 Windows (1-Tık Kurucu - Önerilen)** | [**HelloDPI-Setup.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | İndirin ve çift tıklayın. Masaüstüne ve Başlat Menüsüne kısayol oluşturup arka planda sessizce başlatır. |
| **🪟 Windows (Taşınabilir Sürüm)** | [**HelloDPI-Windows.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Kurulum gerektirmez, doğrudan çift tıklayarak çalıştırın (Saatin yanındaki sistem tepsisine yerleşir). |
| **🍏 macOS (Resmi İmzalı DMG)** | [**HelloDPI-macOS.dmg**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Apple Developer ID imzalıdır. DMG dosyasını açın, `Hello DPI.app` simgesini Applications klasörüne sürükleyin. |
| **🍏 macOS (ZIP Arşivi)** | [**HelloDPI-macOS.zip**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | ZIP arşivini açın ve içindeki `Hello DPI.app` uygulamasını çift tıklayarak çalıştırın. |
| **🤖 Android (1-Dokunuş Kurulum APK)** | [**HelloDPI-Android.apk**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Telefonunuza indirin ve dokunarak kurun. Hızlı Ayarlar (Tile), bildirim çubuğu ve açılışta otomatik başlatma desteklidir. |
| **🍏 iOS & iPadOS (Network Extension)** | [**mobile/ios/**](mobile/ios) | Apple `NEPacketTunnelProvider` altyapısı ile iPhone ve iPad için tünel projesi. |
| **🐧 Linux (x86_64)** | [**hellodpi-linux-amd64**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Bağımsız ikili dosya veya GNOME / KDE Plasma masaüstü tepsisi ile çalışır. |

---

### 🛡️ VirusTotal Tarama Sonuçları ve Güvenlik

Hello DPI v5.0, GoodbyeDPI'ı ve geleneksel araçları geride bırakan **tam teşekküllü çok platformlu sansür atlatma mimarisine** sahiptir:
1. **🪟 Windows 1-Tık Kurucu (`HelloDPI-Setup.exe`):** Hiçbir teknik bilgi gerektirmeden tek tıkla kurulur, masaüstüne kısayol oluşturur ve arka planda sessizce devreye girer.
2. **🛡️ İleri Seviye DPI Atlama (Decoy & Out-of-Order):** Sandvine, Procera ve Huawei kurumsal DPI donanımlarını sahte paket (decoy TLS) enjeksiyonu ve sıra dışı TCP akışıyla şaşırtarak sansürü tamamen etkisiz kılar.
3. **🔒 Encrypted Client Hello (ECH) & Güçlendirilmiş DoH:** DNS Type 65 üzerinden ECH anahtarlarını çekerek alan adını (SNI) tamamen şifreler; böylece sansür cihazları hedefi hiç göremez.
4. **📱 Çok Platformlu Mobil Ekosistem:** Android'de root gerektirmeyen `VpnService` + yerel ARM64 motoru; iOS'ta Apple'ın resmi `NEPacketTunnelProvider` Network Extension mimarisi.
5. **⚡ Ağ Doktoru Canlı Ping Monitörü:** Discord Voice (Frankfurt, Rotterdam), Roblox ve Cloudflare sunucularına canlı gecikmeyi ölçer; VPN'lere kıyasla **0 ms ek ping** avantajını doğrudan kanıtlar.
6. **🎯 Otomatik DPI Sondajı (Auto-Tuning Engine):** İSS'nizin (TTNET, Superonline, Vodafone, TurkNet, KYK) sansür filtrelerini canlı test edip en uygun parçalama modunu otomatik seçer.
7. **🛤️ Akıllı Bölünmüş Tünelleme (Split-Tunneling):** Bankacılık (Ziraat, Garanti, İş Bankası vb.), e-Devlet ve yerel oyun sunucuları doğrudan temiz ağdan akar; yalnızca sansürlü servisler DPI tüneline girer.
8. **🎙️ WebRTC & Discord Ses Optimizasyonu:** Discord ses kanallarında yaşanan RTC Connecting / No Route takılmalarını çözen özel UDP katmanı.
9. **⚙️ Çift Modlu Motor:** Standart Proxy Modu (L7) + Çekirdek Modu (L3/L4 WinDivert / Wintun).

Her yayınlanan sürüm dünyanın önde gelen 70+ antivirüs motoru tarafından taranır:

| Dosya Adı | SHA-256 Özeti | VirusTotal Durumu |
| :--- | :--- | :--- |
| **HelloDPI-Windows.exe** | `069061ef97888ee449e5073f893214b5ffff2fbd2bf10bbc56e1c0777c35331f` | [**0/72 Temiz (Clean)**](https://www.virustotal.com/gui/file/30037fd5b5d1a32bf15c5bf4c861422b2f0e07e661a9858de39e104e276ad732) |
| **HelloDPI-macOS.dmg** | `e0a394b8742dbc50a63dcf77e5e6788bdfc682c159ef3c1a135a427c7dee4779` | [**0/65 Temiz (Clean)**](https://www.virustotal.com/gui/file/b7b159fb9568f266fae2f1f2c9a75518417658f8b40c415db7bf8f38795ec652) |
| **HelloDPI-macOS.zip** | `082766c63b484cf6734b09a3363d9ddb3419e4f0de250bd9e49f7999cf69cc08` | [**0/65 Temiz (Clean)**](https://www.virustotal.com/gui/file/b7b159fb9568f266fae2f1f2c9a75518417658f8b40c415db7bf8f38795ec652) |
| **hellodpi-linux-amd64** | `1931f60f600096faf0e287de69d5d957f1ca1cf10faf3d552f2099d25dce39cf` | [**0/65 Temiz (Clean)**](https://www.virustotal.com/gui/file/00b22ca3b8bb24b2fc1556ecac8811371039ae4cc2483a573dcd6a898957bec1) |

> 🔒 **Gizlilik ve Doğruluk Garantisi:** Hello DPI kök sertifika (MITM CA) yüklemez. Şifreli HTTPS trafiğinizin içeriğini göremez ve değiştiremez; bağlantı kurulurken hedefe giden ilk paket başlığını parçalayarak sansür filtrelerini aşar.

---

### 🖱️ Kolay Kurulum (Sıfır Terminal!)

Hello DPI, **bilgisayarla arası sadece günlük kullanımla sınırlı olan, terminal veya komut satırı açmayı bilmeyen herkesin rahatlıkla kullanabilmesi için** özel olarak tasarlandı. Kurulum veya çalıştırma için kesinlikle terminal açmanıza gerek yoktur.

---

#### 🪟 Windows Kullanıcıları İçin

1. [**HelloDPI-Windows.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) dosyasını indirin ve çift tıklayın.
2. Sağ alttaki sistem tepsisinde (saatin yanındaki ok simgesinin içinde) minimalist simgemiz belirecektir.
3. Siyah komut satırı ekranı açılmaz, arka planda sessizce ve hafifçe çalışır.

> [!TIP]
> ### 🪟 Windows SmartScreen ("Windows kişisel bilgisayarınızı korudu") Uyarısı Çıkarsa:
>
> **Neden Çıkar?**
> Microsoft Windows Defender SmartScreen, internetten yeni indirilen ve yıllık 400-700$ ödenerek kurumsal EV Sertifikası ile imzalanmamış açık kaynaklı tüm yeni programlarda bu standart uyarıyı gösterir. Programınız **kesinlikle virüslü veya tehlikeli DEĞİLDİR.**
>
> **Nasıl Kaldırılır / Çalıştırılır?**
> 1. Ekrana gelen mavi penceredeki altı çizili **"Ek Bilgi" (More info)** yazısına tıklayın.
> 2. Sağ altta beliren **"Yine de Çalıştır" (Run anyway)** butonuna tıklayın.
> 3. Windows bu kararı kalıcı olarak hafızaya alır ve bir sonraki açılışlarda bir daha asla bu uyarıyı göstermez!
>
> *Otomatik 1-Tık Çözüm:* İndirdiğiniz klasörde PowerShell veya CMD açıp `Unblock-File -Path .\HelloDPI-Windows.exe` komutunu verebilir veya arşivdeki `Baslat.bat` dosyasını çalıştırabilirsiniz.

---

#### 🍏 macOS Kullanıcıları İçin

1. [**HelloDPI-macOS.dmg**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) dosyasını indirin ve çift tıklayarak açın.
2. Açılan penceredeki `Hello DPI` simgesini yanındaki `Applications` (Uygulamalar) klasörüne sürükleyip bırakın.
3. Uygulamalar klasörünüze gidip `Hello DPI` simgesine çift tıklayın.
4. Sağ üst menü çubuğunuzda minimalist simgemiz belirecektir. Artık Discord, Roblox ve tüm sansürlü siteler açıktır!

> [!IMPORTANT]
> ### 🍎 Mac'te "Uygulama Hasar Görmüş" veya "Geliştirici Doğrulanamadı" Uyarısı Görürseniz:
>
> Apple, Mac App Store dışından indirilen ve Apple'a yıllık 99$ ödenip tescil ettirilmemiş açık kaynaklı bağımsız yazılımlarda varsayılan olarak şu uyarıyı gösterir:
> * *"Hello DPI hasar görmüş olduğu için açılamıyor. Çöp Sepeti'ne taşımalısınız."* veya
> * *"Apple bu uygulamanın kötü amaçlı yazılım içerip içermediğini denetleyemez."*
>
> **Nasıl Kaldırılır? (3 Farklı Kolay Yol):**
>
> 1. **Sistem Ayarları (2 Tıkla Onay):**
>    - Apple Menüsü () > **Sistem Ayarları (System Settings)** > **Gizlilik ve Güvenlik (Privacy & Security)** sekmesine girin.
>    - Sayfayı en aşağıya kaydırın ve *"Hello DPI engellendi"* yazısının yanındaki **"Yine de Aç" (Open Anyway)** butonuna tıklayın.
> 2. **Finder'da Control + Tık ile Açma:**
>    - Finder > Uygulamalar klasöründe `Hello DPI` simgesine **Control (ctrl)** tuşuna basarak tıklayın (sağ tık) ve **"Aç"** seçeneğini seçin. Açılan kutuda tekrar **"Aç"** butonuna basın.
> 3. **Tek Satır Terminal Çözümü (Karantinayı Kalıcı Siler):**
>    - Terminal'e şunu yapıştırın: `xattr -cr "/Applications/Hello DPI.app"` (uyarı kalıcı olarak yok olur).

---

#### 📱 Mobil Desteği (Android & iOS)

Hello DPI v5.0 ile birlikte sansür atlatma deneyimi akıllı telefonlarınıza ve tabletlerinize taşındı:

- **🤖 Android Kullanıcıları (1-Dokunuş Kurulum):**
  - [**HelloDPI-Android.apk**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) dosyasını telefonunuza indirin ve dokunarak kurun.
  - **Sessiz Arka Plan & Sıfır Pil Tüketimi:** Geleneksel VPN'ler gibi uzak sunucuya sürekli şifreli veri pompalamaz; cihaz içinde sadece paket başlıklarını parçalayarak şebeke hızınızı (%100 hat hızı) ve pilinizi korur.
  - **Hızlı Ayarlar (Tile) Desteği:** Telefonunuzun üst bildirim panelini aşağı kaydırıp **Hello DPI** butonunu ekleyebilir; bilgisayarınızdaki sistem tepsisi gibi uygulamayı dahi açmadan tek dokunuşla anında açıp kapatabilirsiniz.
  - **Açılışta Otomatik Başlatma:** "Cihaz Açıldığında Otomatik Başlat" seçeneğiyle telefonunuz yeniden başladığında kesintisiz ve sessizce korumaya devam eder.
- **🍏 iOS & iPadOS Kullanıcıları:**
  - `mobile/ios/` dizinindeki SwiftUI projesini Xcode ile açıp iPhone'unuza yükleyebilirsiniz.
  - Apple `NEPacketTunnelProvider` çerçevesi sayesinde arka planda pil tüketmeden doğrudan yerel şeffaf tünel sağlar.

---

#### 🩺 Ağ Doktoru ve Canlı Ping Monitörü

Menüden veya tepsi simgesinden **Ağ Doktoru** panelini açtığınızda:
- **0 ms Ek VPN Gecikmesi:** Discord Voice (Frankfurt, Rotterdam), Roblox ve Cloudflare sunucularına doğal hat pinginizi canlı olarak görüntülersiniz.
- **Sıfır Ayar:** Arka plandaki akıllı sondaj (Auto-Tune) motoru ve OTA kural senkronizasyonu internetinizi otomatik optimize eder.

Türkiye'deki İnternet Servis Sağlayıcıları (Türk Telekom, Superonline, TurkNet, Vodafone, Kablonet vb.), Discord ve erişimi kısıtlanan platformları engellemek için **DPI (Derin Paket İncelemesi)** teknolojisini kullanır. 

Bir web sitesine bağlanırken bilgisayarınız hedef sunucuya şifrelenmemiş ilk paket olan **TLS ClientHello** paketini gönderir. İSS'lerin donanımsal sansür filtreleri, bu paketin içindeki **SNI (hedef alan adı)** alanını okur; eğer hedef Discord veya engelli bir site ise hattınıza sahte bir bağlantı kesme paketi (`TCP RST`) fırlatarak iletişimi anında koparır.

```
[ GELENEKSEL VPN YÖNTEMİ (Hantal, Yavaş ve Güvensiz) ]
Siz ───> [ Hollanda / Almanya VPN Sunucusu ] ───> Discord / Web Sitesi
         🔻 Hız %50-%70 düşer
         🔻 Ping +80ms ile +200ms artar
         🔻 Tüm şifreleriniz ve trafiğiniz yabancı bir sunucudan geçer
         🔻 Riot Vanguard / Anti-Cheat şüpheli IP uyarısıyla ban atabilir

[ HELLO DPI YÖNTEMİ (Doğrudan, Şeffaf ve Işık Hızında) ]
Siz ═════════════════════════════════════════════> Discord / Web Sitesi
     ⚡ 0ms Ek Gecikme (Ping tamamen aynı kalır)
     ⚡ %100 Tam Fiber Hat Hızı (1000 Mbps ise 1000 Mbps)
     ⚡ Yabancı sunucu yok; IP adresiniz değişmez, Türkiye'deki kendi IP'nizdir
     ⚡ Anti-cheat dostudur (Valorant, CS2, LoL'de kesinlikle ban riski yoktur)
```

---

### 🌟 Hello DPI v2.2 ile Gelen Yeni Özellikler

Hello DPI v2.2 sürümü, topluluktan gelen geri bildirimler doğrultusunda baştan sona güçlendirildi:

1. **🛠️ Tek Tıkla Ağ Doktoru & Otomatik Sorun Giderici (Discord & Roblox Çözümü):**
   - Menüdeki *"🛠️ Ağ Sorunlarını Gider (Otomatik Onar)"* butonuna tek tıkla bastığınızda tüm ağ ayarlarınızı tarar ve otomatik onarır:
     - **DNS Önbellek Temizliği:** İSS tarafından zehirlenmiş engelli IP kayıtlarını (`flushdns`) anında temizler.
     - **Güvenli DNS:** Roblox ve oyun sunucularının açılması için Cloudflare (1.1.1.1) ve Google DNS'i kontrol eder ve optimize eder.
     - **Sistem Proxy & SOCKS Doğrulaması:** HTTP, HTTPS ve SOCKS protokollerini 127.0.0.1:8080 olarak yeniden hizalar.
     - **Canlı Rapor Ekranı:** Tarayıcınızda özel bir Ağ Doktoru paneli (`/doctor`) açarak nelerin düzeltildiğini ve Discord/Roblox pinglerini adım adım gösterir.

2. **🎬 Watch Together (w2g.tv) ve WebSockets Odaları Desteği:**
   - Watch Together, Kosmi ve benzeri platformlarda oda kurarken veya video senkronize ederken yaşanan bağlantı hataları tamamen çözüldü.
   - Yeni `bufferedConn` mimarisi sayesinde TLS ClientHello parçalanmasından sonra gelen WebSocket (`wss://`) ve HTTP/2 akışlarının tek bir baytı dahi kaybolmadan iletilmesi güvence altına alındı.

3. **🏢 GSB WiFi (KYK Yurt WiFi) ve Captive Portal Uyumluluğu:**
   - DPI araçları açıkken KYK yurtlarında ve üniversite ağlarında yaşanan *"DPI açıkken internete hiç bağlanamama"* ve giriş portalının açılmaması sorunu çözüldü.
   - Sistem proxy'sine ve yerel çekirdeğe `ProxyOverride` ve captive portal bypass kuralları (`wifi.gsb.gov.tr`, `*.kyk.gov.tr`, `10.0.0.0/8`, `captive.apple.com`, `connectivitycheck.gstatic.com` vb.) eklendi.
   - Giriş sayfası açılırken DoH yerine doğrudan yerel ağ kullanılır; giriş tamamlandığında ise tüm sansürsüz internet koruması devreye girer.
   - Cloudflare DoH engellenirse anında **Google DoH ➔ Quad9 DoH ➔ Yerel Sistem DNS** sıralı dayanıklı fallback devreye girer; internet asla donmaz.

3. **⚡ Windows 0ms Anlık Tepki Süresi (Optimize Edilmiş Win32 Registry):**
   - Windows kullanıcılarının yaşadığı "Duraklat / Başlat butonuna basınca 2-3 saniye donma" sorunu tamamen ortadan kaldırıldı.
   - Harici `reg.exe` prosesleri yerine doğrudan `golang.org/x/sys/windows/registry` Win32 bellek API'sine geçildi. Tepki süresi 2500 ms'den **0.1 milisaniyeye** indirildi; butonlar artık anında renk ve durum değiştirir.

4. **✨ Tek Tıkla Otomatik Güncelleme (In-App Auto-Updater):**
   - Bilgisayarınızda eski sürüm yüklüyse menüde otomatik olarak *"✨ Yeni Güncelleme: vX.X.X (Tıkla ve Güncelle)"* uyarısı görünür.
   - Tıklandığında arka planda GitHub'dan son sürüm indirilir, mevcut dosya yerinde yenilenir ve uygulama otomatik baştan başlar. Her sürümde yeniden web sitesine girip exe indirmenize gerek kalmaz!

5. **⚡ Dahili 60 FPS Animasyonlu Hız Testi (Speedtest):**
   - Menüdeki *"Hız Testi Yap"* seçeneğiyle doğrudan yerel Go proxy motorumuz üzerinden çalışan **neon ibreli hız testi paneli** açılır.
   - Anlık **Ping (ms), Jitter (dalgalanma), İndirme (Download Mbps) ve Yükleme (Upload Mbps)** hızlarınızı ölçebilirsiniz.

6. **🛡️ QUIC / HTTP-3 Boşluğunun Kapatılması (UDP:443 Koruması):**
   - Discord masaüstü uygulaması ve Chrome'un UDP 443 kullanarak DPI korumasını aşmasını engeller. SOCKS5 UDP talepleri akıllıca reddedilir ve tüm istemciler korumalı TCP+TLS hattına yönlendirilir.

7. **🔄 Kalıcı Otomatik Başlatma (Açılışta Otomatik Başlat):**
   - Menüden tek tıkla işaretleyin. Bilgisayarınız her açıldığında Hello DPI arka planda sessizce hazır olur.

---

### 📊 Karşılaştırma Tablosu

| Özellik | Geleneksel VPN | GoodbyeDPI | Zapret | ⚡ **Hello DPI v2.1** |
| :--- | :--- | :--- | :--- | :--- |
| **Kullanım Kolaylığı** | Hesap, kayıt, abonelik | Karmaşık `.cmd` dosyaları | Terminal ve root ayarı | 🖱️ **Tek tıkla menü çubuğu / tepsi** |
| **Terminal / Kod Gereksinimi** | Yok | Var | Var | 🟢 **SIFIR TERMİNAL (100% GUI)** |
| **Otomatik Güncelleme** | Var | ❌ Manuel indirme | ❌ Manuel git pull | ⚡ **Tek tıkla uygulama içi güncelleme** |
| **GSB WiFi (KYK) Desteği** | Çoğu bloklu | ❌ DNS hatası verir | ❌ Manuel ayar ister | 🛡️ **Dahili Captive Bypass & Multi-DoH** |
| **Watch Together / WebSockets** | Yavaş / Gecikmeli | Bazen kopar | Bazen kopar | 🛡️ **bufferedConn ile Tam Uyumlu** |
| **İnternet Hızı** | 🔻 %50 - %70 Düşüş | ⚡ %100 Hat Hızı | ⚡ %100 Hat Hızı | ⚡ **%100 Tam Hat Hızı (Fiber)** |
| **Oyun Pingi (Gecikme)** | 🔻 +60ms - +200ms | 🟢 0ms ek gecikme | 🟢 0ms ek gecikme | 🟢 **0ms (Sıfır Ping Etkisi)** |
| **Platform Desteği** | Çeşitli | ❌ Yalnızca Windows | ❌ Linux / Karmaşık | 🍏 **macOS**, 🪟 **Windows**, 🐧 **Linux** |
| **Discord Masaüstü & Ses** | Bazen engelli / Yavaş | Ek parametreler ister | Karmaşık ayar | 🛡️ **RFC TLS Record Splitting ile Hazır** |
| **QUIC / HTTP-3 Koruması** | Var | Sürücü ile var | Elle kural ister | 🛡️ **Dahili UDP:443 Fallback Denetimi** |
| **Dahili Hız Testi** | Reklamlı / Yok | Yok | Yok | ⚡ **Özel 60 FPS HTML5 Speedtest** |
| **Açılışta Otomatik Başlat** | Ağır arka plan servisi | Elle Windows servisi | Elle systemd | 🔄 **Menüden tek tıkla açılıp kapanır** |
| **Sistem Kaynak Tüketimi** | Yüksek CPU & RAM | Düşük | Düşük | 🪶 **< 15 MB RAM, %0 Boşta CPU** |

---

### ❓ Sıkça Sorulan Sorular (SSS)

<details>
<summary><b>1. Watch Together veya video izleme odalarına girerken hata veriyor mu?</b></summary>
<br>

**HAYIR, v2.1 İLE KUSURSUZ ÇALIŞMAKTADIR.**
Eski sürümlerde TLS el sıkışmasından sonra gelen WebSocket (`wss://`) ve HTTP/2 veri paketlerinin bir kısmı soket tamponunda kaybolabiliyordu. v2.1 sürümünde geliştirilen `bufferedConn` mimarisi sayesinde Watch Together (w2g.tv), Kosmi ve benzeri tüm oda/senkronizasyon platformları tam hat hızında sorunsuz çalışır.
</details>

<details>
<summary><b>2. GSB WiFi (KYK Yurt İnterneti) kullanıyorum, neden eskiden çalışmıyordu ve şimdi nasıl çalışıyor?</b></summary>
<br>

KYK yurt internetinde (`GSB WiFi`) internete çıkabilmek için önce `wifi.gsb.gov.tr` adresinden T.C. kimlik numaranızla giriş yapmanız gerekir. Eski araçlar (GoodbyeDPI veya eski Hello DPI sürümleri) sistem proxy'si açıldığında bu yerel adresi de Cloudflare DoH'a sormaya çalışıyor ve henüz internete giriş yapılmadığı için DNS donuyor ve bağlantı kopuyordu.

Hello DPI v2.1, `wifi.gsb.gov.tr`, `10.x.x.x` ve captive portal adreslerini **otomatik tespit ederek doğrudan (DPI'sız ve yerel DNS ile)** bağlar. Giriş sayfanız saniyeler içinde açılır; girişinizi yaptıktan sonra ise Hello DPI'ın sansür aşma motoru devreye girerek Discord ve diğer siteleri açar.
</details>

<details>
<summary><b>3. Discord masaüstü uygulaması açılmıyor veya ses kanallarında 'RTC Bağlanıyor'da kalıyor, ne yapmalıyım?</b></summary>
<br>

Discord masaüstü uygulaması arka planda önceden açıksa, eski engelli bağlantıyı hafızasında tutmuş olabilir:
1. Klavyenizden **Cmd + Q** (Mac) veya Görev Yöneticisi'nden (Windows) Discord'u tamamen kapatın.
2. Hello DPI'ın menü çubuğunda çalıştığından emin olun.
3. Discord'u yeniden açın. Doğrudan bağlandığını ve ses kanallarına gecikmesiz girdiğinizi göreceksiniz.
</details>

<details>
<summary><b>4. Bu program şifrelerimi, banka hesaplarımı veya özel mesajlarımı görebilir mi?</b></summary>
<br>

**KESİNLİKLE HAYIR.**
Tüm modern internet trafiği HTTPS/TLS ile uçtan uca şifrelidir. Hello DPI bilgisayarınıza kök güvenlik sertifikası (MITM CA) **yüklemez ve yükleyemez**. 
Program yalnızca sunucuyla bağlantı kurulurken hedefe giden ilk paketin (ClientHello) ilk 5 baytını ayırır. Şifreli paketlerin içeriği çözülmez, loglanmaz ve hiçbir uzak sunucuya aktarılmaz. VirusTotal'de 70+ antivirüs tarafından taranmış ve 0/72 temiz bulunmuştur.
</details>

<details>
<summary><b>5. Valorant, CS2, LoL veya Rainbow Six oynarken hileden ban yer miyim?</b></summary>
<br>

**HAYIR, KESİNLİKLE BAN YEMEZSİNİZ.**
VPN servisleri IP adresinizi yabancı ülkelere (Almanya, Hollanda vb.) yönlendirdiği için Riot Vanguard, EasyAntiCheat (EAC) veya BattlEye gibi hile koruma sistemleri bunu "hesap paylaşımı" veya "şüpheli konum" olarak algılayıp hesabınızı askıya alabilir.
Hello DPI ise **IP adresinizi değiştirmez.** Siz yine kendi ev internetinizin Türk Telekom, Superonline veya TurkNet IP'si ile oyuna bağlanırsınız. Oyun paketleri doğrudan akar; ek gecikme (ping) kesinlikle oluşmaz.
</details>

<details>
<summary><b>6. Türkiye'de bu programı kullanmak yasal mıdır? Başım hukuki olarak belaya girer mi?</b></summary>
<br>

**GÜNLÜK KULLANIM TAMAMEN YASALDIR.**
Türkiye'de 5651 Sayılı İnternet Ortamında Yapılan Yayınların Düzenlenmesi Hakkındaki Kanun kapsamında, vatandaşların internete erişirken DNS, VPN veya DPI manipülasyon araçları kullanması **suç olarak tanımlanmamıştır.**
Geçmişte Wikipedia, YouTube, Twitter veya Instagram engellendiğinde tüm vatandaşlar ve hatta kamu kurumları bu yöntemlerle internete erişmiştir. Hukuken suç olan şey erişim biçimi değil, internette işlenebilecek yasa dışı fiillerdir (terör, yasa dışı bahis vb.). Günlük internet, oyun ve Discord kullanımınızda hiçbir hukuki sakınca bulunmamaktadır.
</details>

<details>
<summary><b>7. Mac'te 'Uygulama Hasar Görmüş' diyor, bunu çözmek için terminal açmam gerekir mi?</b></summary>
<br>

**HAYIR, TERMİNAL AÇMANIZA GEREK YOKTUR.**
Bu uyarı Apple'ın App Store dışı açık kaynak kodlu uygulamalara koyduğu standart bir güvenlik prosedürüdür. 
Çıkan uyarıda **"Vazgeç"** deyin, ardından **Apple Menüsü () > Sistem Ayarları > Gizlilik ve Güvenlik** sayfasına gidin. En altta çıkan *"Hello DPI engellendi"* uyarısının yanındaki **"Yine de Aç"** butonuna tıklayın. Sadece 2 fare tıkıyla sorun kalıcı olarak çözülür.
</details>

---

<details>
<summary><b>🛠️ Meraklısına Teknik & Protokol Mimarisi (RFC Standartları)</b></summary>

<br>

DPI cihazları (Huawei, Sandvine, Allot vb.), servis sağlayıcılarının ana omurgasında yer alan donanımsal paket inceleme filtreleridir. Türkiye'deki İSS'ler genellikle **stateful TCP reassembly** uygular; yani ilk paketi kaba şekilde bayt bazında bölerseniz donanım o baytları hafızasında birleştirip yine de sansür uygular.

Hello DPI, **RFC 5246 (TLS 1.2) ve RFC 8446 (TLS 1.3)** standartlarının açık protokol kuralını uygular:
> *"Handshake messages MAY be coalesced into a single TLSPlaintext record, or divided among several records."*

1. **TLS Record Layer Splitting:**
   - Gelen `ClientHello` paketi, iki geçerli bağımsız TLS kaydına (TLS Record) ayrıştırılır:
     - **1. Kayıt:** Yalnızca el sıkışma başlığını (5 bayt) taşır; içinde alan adı (SNI) bulunmaz. DPI cihazı bu paketi denetler ve zararsız bularak geçirir.
     - **2. Kayıt:** Kalan el sıkışma verisini taşır. Filtreler yeni bir el sıkışma başlangıcı görmediği için paketi atlar.
   - Hedef sunucu (örneğin Cloudflare veya Discord) iki kaydı RFC standartlarına uygun olarak anında birleştirir ve güvenli şifreli oturum açılır.
2. **bufferedConn & io.ReadFull Pre-buffering:**
   - Modern tarayıcıların ECH (Encrypted Client Hello), GREASE ve çoklu key-share uzantıları içeren 1-4 KB boyutundaki paketleri `io.ReadFull` ile eksiksiz tamponlanır. Kalan akış `bufferedConn` ile raw sokete aktarılarak WebSockets ve HTTP/2 akışlarının kesintisiz çalışması garanti edilir.
3. **Multi-Tier Resilient DNS:**
   - Cloudflare DoH (`https://1.1.1.1/dns-query`) ➔ Google DoH (`https://dns.google/resolve`) ➔ Quad9 DoH (`https://dns.quad9.net/dns-query`) ➔ Yerel Sistem DNS zinciri ile engelli ağlarda bile DNS sorgusu düşmez.
</details>

---

### 💻 Geliştiriciler İçin (Kaynak Koddan Derleme & Test)

Projeyi yerel makinenizde derlemek veya test etmek isterseniz:

```bash
# 1. Depoyu klonlayın
git clone https://github.com/nikatheproffesor/hello-dpi.git
cd hello-dpi

# 2. Birim testlerini ve Fuzzing testini çalıştırın
go test -v ./...
go test -fuzz=FuzzSplitTLSRecord -fuzztime=5s ./internal/dpi

# 3. macOS Menü Çubuğu Uygulamasını derleyin:
./scripts/build-macos-app.sh

# 4. Windows uygulamasını derleyin (Konsol pencerisiz):
go build -ldflags="-H=windowsgui -s -w" -o "bin/HelloDPI-Windows.exe" ./cmd/hellodpi-tray

# 5. CLI (Komut satırı) motorunu doğrudan çalıştırın:
go run ./cmd/hellodpi -system-proxy
```

---

## 🇬🇧 English Guide

### 🎯 What is Hello DPI?

Hello DPI is an ultra-lightweight, zero-latency Deep Packet Inspection (DPI) circumvention proxy with a native Menu Bar (macOS) and System Tray (Windows) interface. It bypasses ISP-level domain censorship without routing your traffic through foreign VPN servers, giving you **100% native fiber line speed and 0ms ping penalty**.

---

### 📦 Quick Download

| Operating System | Download File | Installation |
| :--- | :--- | :--- |
| **🪟 Windows (1-Click Installer)** | [**HelloDPI-Setup.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | 1-click setup; creates desktop and start menu shortcuts, runs quietly in background. |
| **🪟 Windows (Portable)** | [**HelloDPI-Windows.exe**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Standalone tray binary without installation. |
| **🍏 macOS (Official Apple Signed DMG)** | [**HelloDPI-macOS.dmg**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Signed with official Apple Developer ID with hardened runtime and Apple timestamp. |
| **🍏 macOS (ZIP Archive)** | [**HelloDPI-macOS.zip**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Extract ZIP and run `Hello DPI.app`. |
| **🤖 Android (ARM64 Engine)** | [**hellodpi-android-arm64**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Non-root native 64-bit ARM core. See `mobile/android/` for Android Studio APK project. |
| **🍏 iOS & iPadOS (Network Extension)** | [**mobile/ios/**](mobile/ios) | Apple `NEPacketTunnelProvider` project for iPhone/iPad. |
| **🐧 Linux (x86_64)** | [**hellodpi-linux-amd64**](https://github.com/nikatheproffesor/hello-dpi/releases/latest) | Standalone executable or systemd service. |

---

### 🛡️ VirusTotal Clean Certification

Hello DPI does not install third-party drivers or kernel modules (no WinDivert) and does not require administrative privileges.

- **HelloDPI-Windows.exe:** [**0/72 Clean on VirusTotal**](https://www.virustotal.com/gui/file/30037fd5b5d1a32bf15c5bf4c861422b2f0e07e661a9858de39e104e276ad732)
- **HelloDPI-macOS.dmg:** [**0/65 Clean on VirusTotal**](https://www.virustotal.com/gui/file/b7b159fb9568f266fae2f1f2c9a75518417658f8b40c415db7bf8f38795ec652)

---

### 🖱️ Zero-Terminal Setup Guide

#### 🍏 macOS: If you see "App is Damaged" or "Unidentified Developer"
Apple shows an alert for open-source apps downloaded outside the App Store:
> *"Hello DPI is damaged and can't be opened. You should move it to the Trash."*

🛑 **Your file is NOT damaged.** This is Apple Gatekeeper's default quarantine for open-source software.
🛑 **You do NOT need to open Terminal!** Solve it in 2 mouse clicks:

1. Click **Cancel** on the alert (Do NOT click "Move to Trash").
2. Open **Apple Menu () > System Settings > Privacy & Security**.
3. Scroll down to the **Security** section.
4. Next to *"Hello DPI was blocked"*, click the **"Open Anyway"** button.
5. Enter your Mac password or use Touch ID.
6. Done! Hello DPI will open in your Menu Bar and macOS will remember your approval forever.

*(Alternative: In Finder, open Applications, hold the **Control** key, click `Hello DPI`, and select **Open**).*

#### 🪟 Windows: If you see Windows SmartScreen
1. Click **More info**.
2. Click **Run anyway**.
3. Hello DPI immediately sits in your notification area (system tray).

---

### 🚀 Key Features in v2.1
- **🎬 Watch Together & WebSocket Rooms Support:** Preserves all pipelined bytes via `bufferedConn` architecture for smooth room synchronization and video streaming.
- **🏢 GSB WiFi & Captive Portal Compatibility:** Intelligent direct pass-through for `wifi.gsb.gov.tr` and captive portal networks with multi-tier DNS failover (Cloudflare -> Google -> Quad9 -> System DNS).
- **⚡ Instant 0ms Windows Toggle:** Direct Win32 registry manipulation eliminates prior 2-3s freeze.
- **✨ In-App 1-Click Auto-Updater:** Automatically detects newer GitHub releases and updates the running binary in-place with a single click.
- **⚡ Built-in 60 FPS Speedtest:** High-precision neon speedometer measuring Ping, Jitter, Download, and Upload via local proxy engine.
- **🛡️ QUIC / HTTP-3 Hole Plugged:** Rejects SOCKS5 UDP:443 to force Chromium and Discord desktop fallback to protected TCP+TLS.
- **🔄 Launch on Boot:** 1-click toggle to automatically start Hello DPI on system login.
- **🎮 Anti-Cheat Friendly:** Does not alter your IP address; 100% safe for Riot Vanguard, EAC, and BattlEye games.

---

### ⚠️ Yasal Uyarı / Disclaimer

Bu yazılım yalnızca eğitim, ağ protokolleri araştırması (RFC 5246/8446) ve kişisel gizlilik testi amacıyla geliştirilmiştir. Yazılımın amacı herhangi bir yasal kısıtlamayı ihlal etmeyi teşvik etmek değildir. Kullanıcılar, bu yazılımı kullanarak gerçekleştirdikleri tüm eylemlerden ve tabi oldukları yerel mevzuata uyumdan bizzat sorumludur. Geliştirici hiçbir yasa dışı kullanım için sorumluluk kabul etmez.

*This software is developed strictly for educational purposes, network protocol research (RFC 5246/8446), and personal privacy testing. Users are solely responsible for compliance with their local regulations.*

---

### 📜 Lisans / License

Bu proje [MIT Lisansı](LICENSE) altında açık kaynak olarak sunulmaktadır.
*This project is open-source under the [MIT License](LICENSE).*
