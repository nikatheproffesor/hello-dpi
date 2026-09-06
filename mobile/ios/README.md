# Hello DPI - iOS & iPadOS Network Extension Mimarisi

Bu dizin, Hello DPI'ın iOS ve iPadOS üzerinde Apple'ın resmi `NetworkExtension` altyapısı kullanılarak çalışması için gereken Swift projesini içerir.

## Mimari Özellikleri

1. **`NEPacketTunnelProvider` (`PacketTunnel/PacketTunnelProvider.swift`)**:
   - iOS'ta sanal bir TUN arayüzü başlatır.
   - DNS sorgularını doğrudan `1.1.1.1` ve `8.8.8.8` üzerinden şifreli çözer.
   - Tüm TCP/UDP trafiğini sıfır ek gecikmeyle Hello DPI motoruna aktarır.

2. **Minimalist SwiftUI UI (`HelloDPI/ContentView.swift`)**:
   - Saf monokrom (siyah-beyaz) tasarım.
   - Tek dokunuşla tünel bağlantısı açma/kapama (`VPNManager.swift`).

## Xcode'da Çalıştırma ve TestFlight

1. `mobile/ios` dizinini Xcode ile açın.
2. **Signing & Capabilities** sekmesinden kendi Apple Developer Team hesabınızı (`KBGS669D97`) seçin.
3. **+ Capability** butonuna basarak **Network Extensions** -> **Packet Tunnel** kutucuğunu işaretleyin.
4. iPhone veya simülatörünüze tek tıkla derleyin.
