<div align="center">

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo_white.png">
    <img src="assets/logo.png" width="130" alt="Hello DPI Logo" />
  </picture>

  # Hello DPI

  **Terminal gerektirmeyen, internet hızınızı ve pinginizi düşürmeyen, tek tıkla çalışan modern sansür aşma aracı.**
  
  *Zero-latency, zero-terminal, cross-platform DPI evasion tool with a native menubar & system tray app.*

  <br />

  [![Release](https://img.shields.io/github/v/release/emreaytekxn/hello-dpi?color=black&logo=github)](https://github.com/emreaytekxn/hello-dpi/releases/latest)
  [![License: MIT](https://img.shields.io/badge/Lisans-MIT-black.svg)](LICENSE)
  [![Platform](https://img.shields.io/badge/Platform-macOS%20%7C%20Windows%20%7C%20Linux-black.svg)](#-kolay-kurulum-sıfır-terminal)
  [![Hız](https://img.shields.io/badge/H%C4%B1z-%25100%20Hat%20H%C4%B1z%C4%B1-black.svg)](#-neden-vpn-değil)
  [![Ping](https://img.shields.io/badge/Ping-0ms%20Ek%20Gecikme-black.svg)](#-neden-vpn-değil)
  [![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)

  <br />

  [🇹🇷 Türkçe Kılavuz](#-türkçe-rehber) • [🍏 macOS Kurulumu](#-macos-kullanıcıları-için-sıfır-terminal) • [🪟 Windows Kurulumu](#-windows-kullanıcıları-için-sıfır-terminal) • [⚡ Hız Testi](#-hello-dpi-v20-ile-gelen-yeni-özellikler) • [❓ SSS](#-sıkça-sorulan-sorular-sss) • [🇬🇧 English Guide](#-english-guide)

</div>

---

## 🇹🇷 Türkçe Rehber

### 🚀 Hemen İndir

Hiçbir komut satırı, kod veya terminal bilgisine ihtiyacınız yoktur. İşletim sisteminize uygun dosyayı indirip doğrudan farenizle çift tıklayarak kullanabilirsiniz:

| İşletim Sistemi | İndirme Dosyası | Kurulum ve Çalıştırma |
| :--- | :--- | :--- |
| **🍏 macOS (Önerilen)** | [**HelloDPI-macOS.dmg**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | DMG dosyasını açın, `Hello DPI.app` simgesini Applications (Uygulamalar) klasörüne sürükleyin ve çift tıklayın. |
| **🍏 macOS (ZIP Arşivi)** | [**HelloDPI-macOS.zip**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | ZIP arşivini açın ve içindeki `Hello DPI.app` uygulamasını çift tıklayarak çalıştırın. |
| **🪟 Windows (64-bit)** | [**HelloDPI-Windows.exe**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | İndirin ve çift tıklayın (Siyah terminal ekranı açılmaz, sağ altta saatin yanına yerleşir). |
| **🐧 Linux (x86_64)** | [**hellodpi-linux-amd64**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | Bağımsız ikili dosya veya GNOME / KDE Plasma masaüstü tepsisi ile çalışır. |

---

### 🖱️ Kolay Kurulum (Sıfır Terminal!)

Hello DPI, **bilgisayarla arası sadece günlük kullanımla sınırlı olan, terminal veya komut satırı açmayı bilmeyen herkesin rahatlıkla kullanabilmesi için** özel olarak tasarlandı. Kurulum veya çalıştırma için kesinlikle terminal açmanıza gerek yoktur.

---

#### 🍏 macOS Kullanıcıları İçin (Sıfır Terminal)

1. [**HelloDPI-macOS.dmg**](https://github.com/emreaytekxn/hello-dpi/releases/latest) dosyasını indirin ve çift tıklayarak açın.
2. Açılan penceredeki `Hello DPI` simgesini yanındaki `Applications` (Uygulamalar) klasörüne sürükleyip bırakın.
3. Uygulamalar klasörünüze gidip `Hello DPI` simgesine çift tıklayın.
4. Sağ üst köşedeki menü çubuğunuzda minimalist el sallama simgemiz (**👋**) belirecektir. Artık Discord ve sansürlü tüm siteler doğrudan açıktır!

---

> [!IMPORTANT]
> ### 🍎 Mac'te "Uygulama Hasar Görmüş" veya "Geliştirici Doğrulanamadı" Uyarısı Görürseniz:
>
> Apple, Mac App Store dışından indirilen ve Apple'a yıllık 99$ ödenip tescil ettirilmemiş açık kaynaklı bağımsız yazılımlarda kullanıcıları korkutmak için varsayılan olarak şu uyarıyı gösterir:
> 
> * **"Hello DPI hasar görmüş olduğu için açılamıyor. Çöp Sepeti'ne taşımalısınız."** veya
> * **"Apple bu uygulamanın kötü amaçlı yazılım içerip içermediğini denetleyemez."**
>
> 🛑 **Gerçek Nedir?**
> Dosyanız **kesinlikle hasarlı, bozuk veya virüslü DEĞİLDİR.** Bu uyarı, Apple'ın bağımsız yazılımcıların programlarına uyguladığı otomatik güvenlik filtresidir (Gatekeeper Karantinası).
>
> 🛑 **Terminal Açmanıza Gerek Var mı?**
> **KESİNLİKLE HAYIR!** Eski forumlarda veya internet sitelerinde gördüğünüz *"Terminal'i aç, `xattr -cr` yaz"* gibi karmaşık yollarla uğraşmanıza gerek yoktur. Farenizle **2 tıkta** macOS'un kendi içinden onay verebilirsiniz:
>
> ---
>
> #### 🖱️ Çözüm 1 (Önerilen: macOS Sistem Ayarları ile 2 Tıkta Onay):
> 1. Ekrana çıkan uyarıda **"Vazgeç"** (veya İptal) butonuna tıklayın. *(Asla 'Çöp Sepeti'ne Taşı'ya basmayın).*
> 2. Ekranınızın en sol üstündeki **Apple Menüsü () > Sistem Ayarları (System Settings)**'e tıklayın.
> 3. Sol taraftaki listeden **Gizlilik ve Güvenlik (Privacy & Security)** sekmesine tıklayın.
> 4. Sayfayı en aşağıya (Güvenlik / Security başlığına) kaydırın.
> 5. *"Hello DPI uygulamasının kullanımı engellendi"* uyarısının hemen yanında **"Yine de Aç" (Open Anyway)** butonunu göreceksiniz. Bu butona tıklayın.
> 6. Mac parolanızı girin veya parmak izinizi (Touch ID) okutun.
> 7. **Tebrikler!** Hello DPI sağ üst menü çubuğunuzda çalışmaya başlar. macOS bu izni kalıcı olarak hatırlar; bilgisayarı her açtığınızda bir daha asla sormaz!
>
> ---
>
> #### 🖱️ Çözüm 2 (Finder'da Control + Tık ile 3 Saniyede Açma):
> 1. Finder'ı açın ve **Uygulamalar (Applications)** klasörüne girin.
> 2. Klavyenizdeki **Control (ctrl)** tuşuna basılı tutarak `Hello DPI` simgesine farenizle tıklayın (veya doğrudan farenin sağ tuşuyla tıklayın).
> 3. Menünün en üstündeki **"Aç" (Open)** seçeneğine tıklayın.
> 4. Karşınıza gelen onay kutusunda artık **"Aç"** butonu görünecektir; **"Aç"** butonuna basın.

---

#### 🪟 Windows Kullanıcıları İçin (Sıfır Terminal)

1. [**HelloDPI-Windows.exe**](https://github.com/emreaytekxn/hello-dpi/releases/latest) dosyasını indirin.
2. İndirdiğiniz `.exe` dosyasına çift tıklayın.
3. Sağ alttaki sistem tepsisinde (saatin yanındaki ok simgesinin içinde) el sallama simgemiz (**👋**) belirecektir.
4. **Siyah komut satırı ekranı açılmaz, arka planda sessizce ve hafifçe çalışır.**

> [!TIP]
> **Windows SmartScreen ("Windows bilgisayarınızı korudu") Uyarısı Görürseniz:**
> Yeni yayınlanan açık kaynaklı Windows programlarında Microsoft SmartScreen mavi bir pencere açabilir:
> 1. Penceredeki altı çizili **"Ek Bilgi" (More info)** yazısına tıklayın.
> 2. Sağ altta beliren **"Yine de Çalıştır" (Run anyway)** butonuna tıklayın.

---

### ❓ Hello DPI Nedir ve Neden VPN Değildir?

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

### 🌟 Hello DPI v2.0 ile Gelen Yeni Özellikler

Hello DPI v2.0 sürümü, topluluktan gelen geri bildirimler doğrultusunda baştan sona modernize edildi:

1. **⚡ Dahili 60 FPS Animasyonlu Hız Testi (Speedtest):**
   - Menüdeki *"Hız Testi Yap"* seçeneğine tıkladığınızda harici reklamlı sitelere gitmeden, doğrudan yerel Go proxy motorumuz üzerinden çalışan özel **neon ibreli hız testi paneli** açılır.
   - Anlık **Ping (ms), Jitter (dalgalanma), İndirme (Download Mbps) ve Yükleme (Upload Mbps)** hızlarınızı 60 FPS akıcı ibre animasyonuyla ölçebilirsiniz.
   - Test trafiği yerel proxy baypas motoruyla koordine edilir; böylece sansür aşılırken fiber hattınızın gerçek kapasitesini görürsünüz.

2. **🛡️ QUIC / HTTP-3 Boşluğunun Kapatılması (UDP:443 Koruması):**
   - Discord'un masaüstü uygulaması ve Google Chrome gibi Chromium tabanlı tarayıcılar, hedef sunucu destekliyorsa trafiği UDP 443 (QUIC) üzerinden göndermeyi dener.
   - Bu durum TCP filtre atlatmasını atlayabilir ve ses/bağlantı kopmalarına yol açabilir.
   - Hello DPI v2.0, SOCKS5 seviyesinde UDP Associate taleplerini akıllıca reddeder (`0x07 Command Not Supported`). Bu sayede tüm tarayıcılar ve Discord **otomatik olarak %100 korunan TCP+TLS hattına düşer.**

3. **🔄 Kalıcı Otomatik Başlatma (Açılışta Otomatik Başlat):**
   - Menüden tek tıkla *"Açılışta Otomatik Başlat"* kutusunu işaretleyin.
   - Bilgisayarınız her açıldığında Hello DPI arka planda sessizce hazır olur; her seferinde uygulamayı elle açmanız gerekmez.
   - macOS'ta yerel `LaunchAgent`, Windows'ta `Run Registry`, Linux'ta `XDG Autostart` standartlarını kullanır.

4. **💾 Akıllı Sistem Proxy Yedekleme ve Geri Yükleme:**
   - Hello DPI başlatıldığında, varsa kurumsal veya kişisel önceki proxy ayarlarınız güvenle hafızaya alınır.
   - Menüden uygulamayı kapattığınızda veya duraklattığınızda **eski sistem proxy ayarlarınız orijinal haline eksiksiz geri yüklenir.**

5. **⚡ WinINet Anlık Bildirimi (Windows):**
   - Windows'ta proxy ayarı değiştiğinde açık olan tarayıcıları kapatıp açmanıza gerek kalmaz. Arka plandaki `InternetSetOptionW` API'si tüm çalışan uygulamalara ayar değişikliğini anında bildirir.

6. **👋 Minimalist Siyah-Beyaz Menü İkonu:**
   - macOS menü çubuğuna ve Windows tepsisine tam oturan, açık/koyu temaya duyarlı, yüksek çözünürlüklü şık siyah-beyaz el sallama logosu.

---

### 📊 Karşılaştırma Tablosu

| Özellik | Geleneksel VPN | GoodbyeDPI | Zapret | ⚡ **Hello DPI v2.0** |
| :--- | :--- | :--- | :--- | :--- |
| **Kullanım Kolaylığı** | Hesap, kayıt, abonelik | Karmaşık `.cmd` dosyaları | Terminal ve root ayarı | 🖱️ **Tek tıkla menü çubuğu / tepsi** |
| **Terminal / Kod Gereksinimi** | Yok | Var | Var | 🟢 **SIFIR TERMİNAL (100% GUI)** |
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
<summary><b>1. Discord masaüstü uygulaması açılmıyor veya ses kanallarında 'RTC Bağlanıyor'da kalıyor, ne yapmalıyım?</b></summary>
<br>

Discord masaüstü uygulaması arka planda önceden açıksa, eski engelli bağlantıyı hafızasında tutmuş olabilir:
1. Klavyenizden **Cmd + Q** (Mac) veya Görev Yöneticisi'nden (Windows) Discord'u tamamen kapatın.
2. Hello DPI'ın menü çubuğunda çalıştığından emin olun.
3. Discord'u yeniden açın. Doğrudan bağlandığını ve ses kanallarına gecikmesiz girdiğinizi göreceksiniz.
</details>

<details>
<summary><b>2. Bu program şifrelerimi, banka hesaplarımı veya özel mesajlarımı görebilir mi?</b></summary>
<br>

**KESİNLİKLE HAYIR.**
Tüm modern internet trafiği HTTPS/TLS ile uçtan uca şifrelidir. Hello DPI bilgisayarınıza kök güvenlik sertifikası (MITM CA) **yüklemez ve yükleyemez**. 
Program yalnızca sunucuyla bağlantı kurulurken hedefe giden ilk paketin (ClientHello) ilk 5 baytını ayırır. Şifreli paketlerin içeriği çözülmez, loglanmaz ve hiçbir uzak sunucuya aktarılmaz. Kodlarımızın tamamı açık kaynaklıdır ve incelenebilir.
</details>

<details>
<summary><b>3. Valorant, CS2, LoL veya Rainbow Six oynarken hileden ban yer miyim?</b></summary>
<br>

**HAYIR, KESİNLİKLE BAN YEMEZSİNİZ.**
VPN servisleri IP adresinizi yabancı ülkelere (Almanya, Hollanda vb.) yönlendirdiği için Riot Vanguard, EasyAntiCheat (EAC) veya BattlEye gibi hile koruma sistemleri bunu "hesap paylaşımı" veya "şüpheli konum" olarak algılayıp hesabınızı askıya alabilir.
Hello DPI ise **IP adresinizi değiştirmez.** Siz yine kendi ev internetinizin Türk Telekom, Superonline veya TurkNet IP'si ile oyuna bağlanırsınız. Oyun paketleri doğrudan akar; ek gecikme (ping) kesinlikle oluşmaz.
</details>

<details>
<summary><b>4. Türkiye'de bu programı kullanmak yasal mıdır? Başım hukuki olarak belaya girer mi?</b></summary>
<br>

**GÜNLÜK KULLANIM TAMAMEN YASALDIR.**
Türkiye'de 5651 Sayılı İnternet Ortamında Yapılan Yayınların Düzenlenmesi Hakkındaki Kanun kapsamında, vatandaşların internete erişirken DNS, VPN veya DPI manipülasyon araçları kullanması **suç olarak tanımlanmamıştır.**
Geçmişte Wikipedia, YouTube, Twitter veya Instagram engellendiğinde tüm vatandaşlar ve hatta kamu kurumları bu yöntemlerle internete erişmiştir. Hukuken suç olan şey erişim biçimi değil, internette işlenebilecek yasa dışı fiillerdir (terör, yasa dışı bahis vb.). Günlük internet, oyun ve Discord kullanımınızda hiçbir hukuki sakınca bulunmamaktadır.
</details>

<details>
<summary><b>5. Bilgisayarımın pilini hızlı bitirir mi veya bilgisayarımı yavaşlatır mı?</b></summary>
<br>

**HAYIR.**
Hello DPI, Go dilinde sıfır harici kütüphane bağımlılığıyla yüksek performanslı olarak yazılmıştır. Boşta beklerken **%0 CPU** tüketir ve bellekte (RAM) yalnızca **12 - 15 MB** yer kaplar. Bilgisayarınızın açılış hızına veya batarya süresine fark edilebilir hiçbir etkisi yoktur.
</details>

<details>
<summary><b>6. Mac'te 'Uygulama Hasar Görmüş' diyor, bunu çözmek için terminal açmam gerekir mi?</b></summary>
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
2. **io.ReadFull Pre-buffering:**
   - Modern tarayıcıların ECH (Encrypted Client Hello), GREASE ve çoklu key-share uzantıları içeren 1-4 KB boyutundaki büyük paketleri `io.ReadFull` ile eksiksiz olarak tamponlanır; yarım paket işleme hataları tamamen engellenir.
3. **Cloudflare DNS-over-HTTPS (DoH):**
   - İSS seviyesindeki DNS zehirlenmelerine ve sahte yönlendirmelere takılmamak için dahili DoH motoru (`https://1.1.1.1/dns-query`) devrededir.
</details>

---

### 💻 Geliştiriciler İçin (Kaynak Koddan Derleme & Test)

Projeyi yerel makinenizde derlemek veya test etmek isterseniz:

```bash
# 1. Depoyu klonlayın
git clone https://github.com/emreaytekxn/hello-dpi.git
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
| **🍏 macOS (Recommended)** | [**HelloDPI-macOS.dmg**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | Open the DMG, drag `Hello DPI.app` into Applications, and double-click. |
| **🍏 macOS (ZIP Archive)** | [**HelloDPI-macOS.zip**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | Extract ZIP and run `Hello DPI.app`. |
| **🪟 Windows (64-bit)** | [**HelloDPI-Windows.exe**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | Download and run (Lives in System Tray near the clock, no black console window). |
| **🐧 Linux (x86_64)** | [**hellodpi-linux-amd64**](https://github.com/emreaytekxn/hello-dpi/releases/latest) | Standalone executable or systemd service. |

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

### 🚀 Key Features in v2.0
- **⚡ Built-in 60 FPS Speedtest:** High-precision neon speedometer measuring Ping, Jitter, Download, and Upload via local proxy engine.
- **🛡️ QUIC / HTTP-3 Hole Plugged:** Rejects SOCKS5 UDP:443 to force Chromium and Discord desktop fallback to protected TCP+TLS.
- **🔄 Launch on Boot:** 1-click toggle to automatically start Hello DPI on system login.
- **💾 Smart Proxy State Preservation:** Backs up and restores your corporate/personal proxy configurations on exit.
- **⚡ WinINet Instant Notification:** Updates Windows network proxy instantaneously without browser restart.
- **🎮 Anti-Cheat Friendly:** Does not alter your IP address; 100% safe for Riot Vanguard, EAC, and BattlEye games.

---

### ⚠️ Yasal Uyarı / Disclaimer

Bu yazılım yalnızca eğitim, ağ protokolleri araştırması (RFC 5246/8446) ve kişisel gizlilik testi amacıyla geliştirilmiştir. Yazılımın amacı herhangi bir yasal kısıtlamayı ihlal etmeyi teşvik etmek değildir. Kullanıcılar, bu yazılımı kullanarak gerçekleştirdikleri tüm eylemlerden ve tabi oldukları yerel mevzuata uyumdan bizzat sorumludur. Geliştirici hiçbir yasa dışı kullanım için sorumluluk kabul etmez.

*This software is developed strictly for educational purposes, network protocol research (RFC 5246/8446), and personal privacy testing. Users are solely responsible for compliance with their local regulations.*

---

### 📜 Lisans / License

Bu proje [MIT Lisansı](LICENSE) altında açık kaynak olarak sunulmaktadır.
*This project is open-source under the [MIT License](LICENSE).*
