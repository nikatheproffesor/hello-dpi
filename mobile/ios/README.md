# Hello DPI - iOS App Store & TestFlight Dağıtım Rehberi

Bu belge, **Hello DPI iOS** uygulamasını derleme, cihaza yükleme, **TestFlight** ve **Apple App Store**'da yayımlama adımlarını içerir.

---

## 🏗️ Mimari Yapı

Hello DPI iOS uygulaması iki ana bileşenden oluşur:
1. **HelloDPI (Ana SwiftUI Uygulaması):**
   - Saf monokrom (siyah-beyaz) minimalist arayüz.
   - `NETunnelProviderManager` üzerinden sistem VPN profilini kurar ve tek dokunuşla tüneli başlatır/durdurur.
2. **PacketTunnel (Network Extension Target):**
   - `com.apple.networkextension.packet-tunnel` yetkisiyle (entitlement) iOS arka planında bağımsız sandbox sürecinde çalışır.
   - [mobile/ios/Frameworks/HelloCore.xcframework](Frameworks/HelloCore.xcframework) üzerinden saf Go DPI motorunu (`127.0.0.1:18080`) çalıştırır.
   - `NEProxySettings` ve `NEDNSSettings` (DoH 1.1.1.1 / 8.8.8.8) ile tüm iOS HTTP/HTTPS trafiğini sıfır gecikmeyle Hello DPI motoruna yönlendirir.

---

## 🚀 1. Adım: HelloCore.xcframework Üretimi

Mac terminalinizde tek komutla Go çekirdeğini iOS Device + Simulator destekli XCFramework olarak derleyin:

```bash
chmod +x scripts/build-ios-framework.sh
./scripts/build-ios-framework.sh
```
*Bu komut `mobile/ios/Frameworks/HelloCore.xcframework` klasörünü otomatik üretir.*

---

## 📱 2. Adım: Xcode Projesi Oluşturma & Ayarlar

1. **Xcode'u Açın:** `File -> New -> Project` -> **iOS App** seçin.
   - **Product Name:** `HelloDPI`
   - **Interface:** SwiftUI
   - **Language:** Swift
   - **Bundle Identifier:** `com.hellodpi.app`
   - Dosya konumu olarak `mobile/ios` dizinini seçin.

2. **Network Extension Target'ı Ekleyin:**
   - Xcode içinde sol menüden proje kökünü seçin.
   - Alttaki **`+` (Add Target)** butonuna basın.
   - **Network Extension** seçin.
   - **Product Name:** `PacketTunnel`
   - **Provider Type:** `Packet Tunnel`
   - **Language:** Swift
   - **Bundle Identifier:** `com.hellodpi.app.PacketTunnel`

3. **Mevcut Dosyaları Projeye Dahil Edin:**
   - `mobile/ios/HelloDPI/` içindeki `ContentView.swift`, `VPNManager.swift`, `HelloDPIApp.swift` dosyalarını ana target'a ekleyin.
   - `mobile/ios/PacketTunnel/` içindeki `PacketTunnelProvider.swift` dosyasını `PacketTunnel` target'ına ekleyin.
   - `Frameworks/HelloCore.xcframework` klasörünü sürükleyip **PacketTunnel** target'ının **Frameworks and Libraries** bölümüne ekleyin.

4. **Signing & Capabilities (İmzalama):**
   - **HelloDPI target:**
     - `Signing & Capabilities` -> **+ Capability** -> **Network Extensions** -> **Packet Tunnel** kutucuğunu işaretleyin.
   - **PacketTunnel target:**
     - `Signing & Capabilities` -> **+ Capability** -> **Network Extensions** -> **Packet Tunnel** kutucuğunu işaretleyin.
     - **Team:** Kendi Apple Developer Team hesabınızı seçin.

---

## 🧪 3. Adım: Cihazda Test Etme

1. iPhone'unuzu Lightning/USB-C kablosuyla Mac'e bağlayın.
2. Xcode'da hedef cihaz olarak iPhone'unuzu seçin.
3. `Cmd + R` tuşuna basarak uygulamayı çalıştırın.
4. Uygulama açıldığında **"BAĞLAN"** butonuna dokunun.
5. iOS'un standart *"Hello DPI VPN Konfigürasyonu Eklemek İstiyor"* onay bildirimine **İzin Ver** deyin.
6. Durum göstergesi beyaz halkaya dönüştüğünde tünel ve DPI atlatma aktiftir!

---

## 🌐 4. Adım: TestFlight & App Store Dağıtımı

### Gereksinimler:
- Aktif bir **Apple Developer Program** üyeliği ($99/yıl).
- [developer.apple.com](https://developer.apple.com) üzerinde `com.hellodpi.app` ve `com.hellodpi.app.PacketTunnel` App ID'leri (Network Extensions özellikli).

### Adımlar:
1. **Archive Oluşturma:**
   - Hedef cihazı `Any iOS Device (arm64)` olarak seçin.
   - Xcode üst menüsünden `Product -> Archive` seçin.
2. **TestFlight'a Gönderme:**
   - Organizer penceresinde **Distribute App** -> **TestFlight & App Store** seçeneğini işaretleyin.
   - Otomatik imzalama (Automatically manage signing) ile yüklemeyi tamamlayın.
3. **App Store İnceleme Soruları (Guideline 5.4 & Export Compliance):**
   - **Export Compliance (Kriptografi):** *"Does your app use encryption?"* -> **YES** -> *"Is it exempt under standard TLS/HTTPS?"* -> **YES** (Standard TLS muafiyeti seçilir).
   - **Gizlilik Politikası (Privacy Policy):** Hello DPI sıfır kişisel veri (Zero PII) politikasına sahiptir. Kullanıcı trafiği üçüncü bir sunucuya gitmez, yerel localhost'ta işlenir.
