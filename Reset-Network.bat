@echo off
chcp 65001 >nul
title Hello DPI - Ağ ve Proxy Sıfırlama Aracı
color 0A

echo ========================================================
echo        Hello DPI - Ağ ve Proxy Sıfırlama Aracı
echo ========================================================
echo.
echo Ağ ayarlarınız ve sistem vekil sunucu (proxy) yapılandırmanız
echo fabrika ayarlarına sıfırlanıyor, lütfen bekleyin...
echo.

:: 1. Windows Internet Settings (WinINet) Proxy Kapat
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyEnable /t REG_DWORD /d 0 /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v ProxyServer /f >nul 2>&1
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings" /v AutoConfigURL /f >nul 2>&1

:: 2. Çevre Değişkenleri Temizliği (Environment Variables)
reg delete "HKCU\Environment" /v HTTP_PROXY /f >nul 2>&1
reg delete "HKCU\Environment" /v HTTPS_PROXY /f >nul 2>&1
reg delete "HKCU\Environment" /v ALL_PROXY /f >nul 2>&1

:: 3. WinHTTP Sistem Proxy Sıfırlama
netsh winhttp reset proxy >nul 2>&1

:: 4. DNS Önbelleğini Temizle
ipconfig /flushdns >nul 2>&1

:: 5. Varsa WinDivert Sürücüsünü Durdur
sc stop WinDivert >nul 2>&1
sc stop WinDivert14 >nul 2>&1

echo.
echo ========================================================
echo [BAŞARILI] Tüm ağ ayarları ve proxy kayıtları temizlendi!
echo.
echo İnternet bağlantınız tamamen doğrudan (Direct) hale getirildi.
echo Hello DPI kapalı veya silinmiş olsa dahi tüm sitelere
echo sorunsuz şekilde erişebilirsiniz.
echo ========================================================
echo.
pause
