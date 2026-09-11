#!/usr/bin/env bash
# Hello DPI - macOS Ağ ve Proxy Sıfırlama Aracı

echo "========================================================"
echo "       Hello DPI - macOS Ağ ve Proxy Sıfırlama Aracı"
echo "========================================================"
echo ""
echo "macOS vekil sunucu (proxy) ve ağ ayarları sıfırlanıyor..."
echo ""

# 1. Tüm aktif ağ servislerindeki proxy'leri devre dışı bırak
networksetup -listallnetworkservices 2>/dev/null | grep -v "\*" | while IFS= read -r s; do
    [ -z "$s" ] && continue
    echo "Servis kontrol ediliyor: $s"
    networksetup -setwebproxystate "$s" off 2>/dev/null || true
    networksetup -setsecurewebproxystate "$s" off 2>/dev/null || true
    networksetup -setsocksfirewallproxystate "$s" off 2>/dev/null || true
    networksetup -setautoproxystate "$s" off 2>/dev/null || true
done

# 2. launchctl kullanıcı oturumu çevre değişkenlerini temizle
for ev in http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY; do
    launchctl unsetenv "$ev" 2>/dev/null
done

# 3. macOS DNS Önbelleğini temizle
dscacheutil -flushcache 2>/dev/null || true
killall -HUP mDNSResponder 2>/dev/null || true

echo ""
echo "========================================================"
echo "[BAŞARILI] macOS proxy ve ağ ayarları başarıyla sıfırlandı!"
echo ""
echo "İnternet bağlantınız doğrudan (Direct) hale getirildi."
echo "Tüm tarayıcılar ve uygulamalar normale döndü."
echo "========================================================"
echo ""
read -p "Kapatmak için Enter tuşuna basın..."
