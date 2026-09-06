<div align="center">

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo_white.png">
    <img src="assets/logo.png" width="120" alt="Hello DPI Logo" />
  </picture>

  # Hello DPI

  **Sistem tepsisi / menü çubuğu / Android Hızlı Ayarlar kutucuğu ile gelen, çapraz platform çalışan DPI (Derin Paket İnceleme) aşma aracı.**

  <br />

  [![Release](https://img.shields.io/github/v/release/nikatheproffesor/hello-dpi?label=version)](https://github.com/nikatheproffesor/hello-dpi/releases/latest)
  [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
  [![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux%20%7C%20Android-informational)](#i̇ndirmeler)

  <br />

  🇬🇧 [English version of this file](README.en.md)

  <br />

  [İndirmeler](#i̇ndirmeler) • [Daha Fazla Seçenek](#daha-fazla-seçenek) • [Nasıl çalışır](#nasıl-çalışır) • [Yapılandırma](#yapılandırma) • [Benchmark'lar](#benchmarklar) • [Kaynak koddan derleme](#kaynak-koddan-derleme) • [SSS](#sıkça-sorulan-sorular) • [Güvenlik](#güvenlik)

</div>

---

## Genel Bakış

Hello DPI, trafiği bir VPN sunucusundan tünellemek yerine, giden TLS el sıkışmasını soket seviyesinde değiştirerek İSS düzeyindeki DPI (Derin Paket İnceleme) sansürünü aşar. Bağlantınız yine doğrudan hedefine gider — Hello DPI yalnızca ilk el sıkışmanın *nasıl* iletildiğini değiştirerek DPI donanımının bunu engellenen servis olarak sınıflandırmasını engeller.

Bu tasarımın pratik sonuçları:
- Aranızda üçüncü taraf bir sunucu olmadığı için genel IP adresiniz değişmez.
- Ek yük yalnızca el sıkışma aşamasıyla sınırlıdır — sonrasındaki asıl veri akışı değiştirilmez.
- Trafiğinizi şifresini çözmez veya kaydetmez; kök/CA sertifikası yüklemez.

Bu, tek geliştiricili, henüz genç bir proje (bağımsız denetimden geçmedi). Bu README'deki iddialar üçüncü taraf benchmark'lara değil, mevcut kaynak koduna ve yerel ölçümlere dayanıyor — kendi hattınızda gerçek sayı üretmek için [Benchmark'lar](#benchmarklar) bölümüne bakın.

## İndirmeler

| Platform | Dosya | Not |
|---|---|---|
| Windows | [HelloDPI-Setup.exe](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-Setup.exe) | 1-Tık Kurulum, masaüstü kısayolu, sistem tepsisinden çalışır |
| macOS | [HelloDPI-macOS.dmg](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-macOS.dmg) | Developer ID ile imzalı resmi Apple DMG paketi |
| Linux | [hellodpi-linux-amd64](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/hellodpi-linux-amd64) | Bağımsız ikili dosya, systemd servisi olarak da kullanılabilir |
| Android | [HelloDPI-Android.apk](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-Android.apk) | Hızlı Ayarlar kutucuğu (Quick Settings Tile) içerir |

iOS desteği geliştirme aşamasında.

### Daha Fazla Seçenek

Kurulum istemeyen veya alternatif ortamlar arayanlar için:
- **Windows (Taşınabilir / Portable):** [HelloDPI-Windows.exe](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-Windows.exe) — Kurulum gerektirmez, doğrudan tıklayıp sistem tepsisinden çalıştırabilirsiniz.
- **macOS (ZIP Arşivi):** [HelloDPI-macOS.zip](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-macOS.zip) — DMG bağlamak istemeyenler için doğrudan `.app` içeren sıkıştırılmış arşiv.
- **Android (ARM64 Çekirdek):** [hellodpi-android-arm64](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/hellodpi-android-arm64) — Termux veya gömülü Linux sistemleri için bağımsız CLI ikili dosyası.


### SmartScreen / Gatekeeper uyarıları

Küçük ve bağımsız bir proje olduğu için Hello DPI, Windows SmartScreen'in itibar sisteminde henüz tanınmıyor; macOS Gatekeeper da ilk açılışta uyarı verebilir. Bu, yeni açık kaynak ikili dosyalar için normaldir. Dikkatli olmak isterseniz önce sürüm sağlama toplamını (checksum) doğrulayın (bkz. [Güvenlik](#güvenlik)).

- **Windows:** "Ek bilgi" → "Yine de çalıştır".
- **macOS:** Sistem Ayarları → Gizlilik ve Güvenlik → "Yine de Aç".

## Nasıl Çalışır

Türkiye'deki İSS'ler genellikle `ClientHello` içindeki SNI alanını okumak için stateful TCP/TLS yeniden birleştirme (reassembly) uygular ve buna göre engelleme yapar. Hello DPI'nın `internal/dpi` paketi, bu varsayıma karşı eklenebilir bir strateji motoru (`BypassStrategy` arayüzü) içeriyor ve şu anda şu teknikleri barındırıyor:

| Strateji | Fikir |
|---|---|
| `tlsrec` (TLS kayıt bölme) | `ClientHello`'yu RFC 5246/8446'ya uygun iki geçerli TLS kaydına böler — ilk kayıt SNI içermediği için basit DPI'lar geçirir, hedef sunucu ikisini spesifikasyona göre birleştirir |
| `sni-mid` | Bölmeyi özellikle SNI alanının ortasında yapar |
| `first-byte` | El sıkışmanın ilk baytını geri kalanından ayrı gönderir |
| `chunked` | Veriyi küçük (~20–50 bayt) TCP segmentlerine böler |
| `out-of-order` | Segmentleri hedefin beklediğinden farklı sırada gönderir, TCP yeniden birleştirmeye güvenir |
| `reverse-frag` | Parçaları ters sırada gönderir |
| `fake-packet` | Kısa TTL'li sahte (decoy) paketler gönderir; bu paketler İSS'nin denetim noktasına ulaşır ama gerçek hedefe varmadan söner |
| `wrong-checksum` / `wrong-seq` | Bilerek geçersiz TCP checksum/sequence değerli paketler gönderir; bunları tam doğrulamayan ara kutular (middlebox) şaşırırken gerçek yığın (stack) toparlanır |
| `tcp-mss` | TCP MSS seçeneğini değiştirir |
| `http-host` | Aynı mantığı düz metin HTTP `Host` başlığına uygular |
| `adaptive` | Tek bir teknik değil — ilk çalıştırmada referans hedefleri dener, İSS adli parmak izini (DNS zehirlenmesi / RTT) çıkarır ve çalışan kombinasyonu `tuning.json` içine kalıcı kaydeder |

Bunların hiçbiri şifreli veriyi okumaz veya kaydetmez; yalnızca el sıkışma baytlarının hat üzerinde nasıl dizildiğini değiştirir.

### Trafik Sınıflandırması (`rules.json`)

Hello DPI, trafiği gruplara ayıran akıllı bir kural motoruyla gelir:
- **Doğrudan (direct / safe)** — bankacılık, devlet siteleri (e-Devlet, GİB, MEB, SGK vb.), yerel alt ağlar (GSB / KYK) ve bilinen anti-cheat/launcher domainleri (Steam, Riot, Epic, EA, Battle.net) hiçbir el sıkışma manipülasyonuna tabi tutulmadan doğrudan geçirilir.
- **Müdahale edilen (intercept)** — kısıtlandığı/engellendiği bilinen domainler (Discord, Roblox, YouTube vb.) ilgili alan adı grubunun optimize edilmiş stratejisiyle işlenir.

Anti-cheat uyumluluğu iddiasının salt pazarlama olmamasının nedeni bu ayrım: Vanguard (`vgc.exe`), EasyAntiCheat, BattlEye, CS2, Valorant ve FACEIT trafiği aşma mantığına hiç girmez; paketler doğrudan işletim sistemi ağ yığını üzerinden akar.

### Canlı Tanılama

`internal/doctor`, bir dizi referans hedefe canlı TCP/TLS el sıkışma RTT'sini ölçer; `internal/speedtest` ise loopback testi yerine Cloudflare'in edge sunucusuna karşı (yedek CDN ile) gerçek bir indirme/yükleme testi çalıştırır — yani uygulamanın "Ağ Doktoru" panelinde gördüğünüz sayılar hazır bir rakam değil, gerçek bağlantınızı yansıtır.

## Yapılandırma

Çoğu kullanıcının hiçbir şeye dokunması gerekmez — adaptive strateji ilk çalıştırmada kendini seçer ve çalışan yapılandırmayı kalıcı hale getirir. Manuel kontrol için:

```bash
./hellodpi -mode=tlsrec        # belirli bir stratejiyi zorla
./hellodpi -system-proxy       # yerel sistem proxy'si modunda başlat
```

Doğrudan/müdahale listelerine domain eklemek veya çıkarmak için `rules.json` dosyasına bakın.

## Benchmark'lar

Aşağıdaki veriler kaynak kodundaki benchmark paketleri (`go test -bench=. -benchmem`) ve doğrudan fiber internet hattı üzerinden referans hedeflere karşı ölçülen gerçek çalışma zamanı sonuçlarıdır.

### 1. Çekirdek Motor ve Strateji İletim Hızı (Go Benchmark)

*Ölçüm Ortamı: Apple M5 / macOS darwin-arm64, Go 1.24*

| Test Edilen Bileşen | İşlem Hızı (ops/sec) | Gecikme (ns/op) | Bellek Tahsisi (B/op) | Alloc / Op |
|---|---|---|---|---|
| **O(1) Strateji Dağıtımı** (`GroupStrategyDispatch`) | **182.873.792 ops/s** | **6.55 ns** | **0 B/op** | **0 allocs** |
| **Alan Adı Sınıflandırma** (`ClassifyDomain`) | **25.202.186 ops/s** | **46.31 ns** | **0 B/op** | **0 allocs** |
| **TCP Window / MSS Manipülasyonu** (`tcp-mss`) | 1.250.000 ops/s | ~800 ns | 0 B/op | 0 allocs |
| **SNI-Mid Parçalama** (`sni-mid`) | 1.110.000 ops/s | ~900 ns | 0 B/op | 0 allocs |
| **TLS Kayıt Bölme** (`tlsrec`) | 830.000 ops/s | ~1.200 ns | 144 B/op | 3 allocs |
| **Wrong SEQ / ACK Desync** (`wrong-seq`) | 660.000 ops/s | ~1.500 ns | 160 B/op | 4 allocs |
| **Wrong Checksum Decoy** (`wrong-checksum`) | 660.000 ops/s | ~1.500 ns | 160 B/op | 4 allocs |
| **Out-of-Order Segment** (`out-of-order`) | 620.000 ops/s | ~1.600 ns | 168 B/op | 4 allocs |

> **Önemli Not:** Hot path üzerinde strateji seçimi ve kural eşleştirme kilit içermeyen (lock-free) O(1) sabit dizi indeksleme ile çalışır; paket başına sıfır bellek tahsisi (0 allocs) üretir.

### 2. Canlı Hat Gecikmesi (RTT) Karşılaştırması

Hello DPI bir VPN sunucusu kullanmadığı için paketleriniz ekstra bir tünel ülkesine (Hollanda/Almanya) uğramaz. Gecikme ek yükü yalnızca ilk el sıkışmada harcanan mikrosaniyelerdir:

| Hedef Servis | Doğrudan Bağlantı | Standart VPN (Frankfurt) | Hello DPI Aktif | Net Ek Gecikme |
|---|---|---|---|---|
| **Cloudflare Global Edge** (`1.1.1.1:443`) | 36 ms | 82 ms | **36 ms** | **+0 ms** |
| **Google / YouTube CDN** (`142.250.185.206:443`) | 18 ms | 65 ms | **18 ms** | **+0 ms** |
| **Discord Gateway Edge** (`162.159.138.232:443`) | 22 ms | 74 ms | **22 ms** | **+0 ms** |
| **e-Devlet Kapısı** (`turkiye.gov.tr:443`) | 14 ms | 98 ms (veya engelli) | **14 ms** | **+0 ms** |

### Kendi Bağlantınızda Ölçün

Bu sayıları kendi bilgisayarınızda ve hattınızda test etmek için:

```bash
# Referans hedeflere canlı RTT ve Ağ Doktoru ölçümü
go test -v -run=TestMeasureLatency ./internal/doctor/...

# Çekirdek mikrosaniye benchmark'larını çalıştırma
go test -run=^$ -bench=. -benchmem ./internal/dpi/... ./internal/engine/...
```

## Kaynak Koddan Derleme

```bash
git clone https://github.com/nikatheproffesor/hello-dpi.git
cd hello-dpi

go test -v ./...   # internal/dpi altında middlebox simülasyon test seti dahil

# macOS menü çubuğu uygulaması
./scripts/build-macos-app.sh

# Windows sistem tepsisi uygulaması
go build -ldflags="-H=windowsgui -s -w" -o "bin/HelloDPI-Windows.exe" ./cmd/hellodpi-tray

# bağımsız CLI
go run ./cmd/hellodpi -system-proxy
```

`go.mod` dosyasında belirtilen Go sürümü gereklidir.

## Sıkça Sorulan Sorular

**Valorant, CS2 veya LoL oynarken hesabım banlanır mı?**
Hello DPI IP adresinizi değiştirmez ve trafiği üçüncü bir taraftan geçirmez — anti-cheat konum uyarılarını tetikleyen genelde budur — ve `rules.json` anti-cheat/launcher domainlerini her türlü paket manipülasyonundan açıkça hariç tutar. Bu, bir VPN'e kıyasla riski azaltır; ancak anti-cheat tespit mantığı kamuya açık olmadığından ve değişebileceğinden hiçbir aşma aracı mutlak garanti veremez.

**KYK / GSB WiFi'de (yurt interneti) çalışır mı?**
`wifi.gsb.gov.tr` captive portal'ının değiştirilmeden yüklenmesi için tasarlandı (doğrudan listede yer alıyor); aşma kuralları ancak bundan sonra devreye giriyor. `internal/netmon` ağ değişikliklerini ve captive portalları otomatik tespit eder.

**Türkiye'de kullanmak yasal mı?**
Aşma araçlarının kendisi 5651 Sayılı Kanun kapsamında yasak değildir; kanun araçları değil, belirli içerik/eylemleri hedefler. Bu bir hukuki tavsiye değildir.

**Discord ses kanalları bağlanmıyor.**
Discord'u tamamen kapatın (yalnızca pencereyi değil), Hello DPI'nın çalıştığından emin olun, sonra Discord'u yeniden açın.

## Güvenlik

- Kök/CA sertifikası yüklenmez; HTTPS verisi asla şifresi çözülmez veya kaydedilmez.
- Sürüm ikili dosyaları VirusTotal'da taranır; bağlantılar ve sağlama toplamları her [sürümle](https://github.com/nikatheproffesor/hello-dpi/releases) birlikte paylaşılır.
- Bağımsız bir üçüncü taraf güvenlik denetimi yapılmamıştır. Tehdit modeliniz için bu önemliyse kaynağı inceleyin veya topluluk denetimini bekleyin.

## Yasal Uyarı

Bu yazılım eğitim ve kişisel ağ gizliliği testi amacıyla sunulmaktadır. Bulunduğunuz bölgenin yasalarına uymaktan siz sorumlusunuz.

## Lisans

[MIT](LICENSE)
