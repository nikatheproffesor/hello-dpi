# Hello DPI - Android VpnService & Gomobile Entegrasyonu

Bu dizin, Hello DPI'ın mobil çekirdeğini Android platformunda root izni gerektirmeden `VpnService` sanal ağ bağdaştırıcısı üzerinden çalıştırmak için hazırlanmış temiz mimariyi içerir.

## Mimari Özeti

1. **Gomobile Bindings (`pkg/hellocore`)**:
   Hello DPI çekirdeğini Android için `.aar` kütüphanesine dönüştürür:
   ```bash
   go install golang.org/x/mobile/cmd/gomobile@latest
   gomobile init
   gomobile bind -target=android -o mobile/android/app/libs/hellocore.aar ./pkg/hellocore
   ```

2. **Android VpnService (`HelloDpiVpnService.kt`)**:
   - Android sisteminden VPN izinlerini talep eder.
   - Yerel `10.0.0.2` IP'si ve `1500` MTU ile TUN sanal arabirimini açar.
   - DNS sorgularını doğrudan `1.1.1.1` ve `8.8.8.8` üzerinden şifreli çözer.
   - TCP akışlarını doğrudan `hellocore` Go motoruna yönlendirerek tüm sansür ve DPI engellerini mobilde de sıfır ek gecikmeyle aşar.

3. **Minimalist Monokrom UI (`MainActivity.kt`)**:
   - Tek dokunuşla bağlan/kes.
   - 0 ms ek gecikme durumu.
