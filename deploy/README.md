# Hello DPI Deployment Guide (Router & Server)

Bu rehber, **Hello DPI**'ı ev ağınızdaki OpenWrt router'ınızda veya Linux sunucunuzda kalıcı bir servis (daemon) olarak çalıştırma adımlarını içerir.

---

## 1. OpenWrt Router Kurulumu

### Adım 1: Router Mimarisine Göre Derleme
Bilgisayarınızda (Go kurulu olan ortamda) router'ınızın işlemci mimarisine göre tek komutla statik ikiliyi (binary) derleyin:

```bash
# MediaTek / MIPSLE router'lar (ör. Xiaomi Mi Router 4A, TP-Link Archer C6U vb.)
GOOS=linux GOARCH=mipsle GOMIPS=softfloat CGO_ENABLED=0 go build -ldflags="-s -w" -o hellodpi-openwrt ./cmd/hellodpi

# ARM64 router'lar (ör. GL.iNet Flint 2, NanoPi R4S, Raspberry Pi vb.)
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o hellodpi-openwrt ./cmd/hellodpi

# Standart x86_64 OpenWrt (Mini PC / x86 router'lar)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o hellodpi-openwrt ./cmd/hellodpi
```

### Adım 2: Dosyaları Router'a Yükleme
```bash
# Binary'yi kopyala
scp hellodpi-openwrt root@192.168.1.1:/usr/bin/hellodpi
ssh root@192.168.1.1 "chmod +x /usr/bin/hellodpi"

# Init script ve UCI config'i kopyala
scp deploy/openwrt/hellodpi.init root@192.168.1.1:/etc/init.d/hellodpi
ssh root@192.168.1.1 "chmod +x /etc/init.d/hellodpi"
scp deploy/openwrt/hellodpi.config root@192.168.1.1:/etc/config/hellodpi
```

### Adım 3: Servisi Başlatma ve Otomatik Açılış
```bash
ssh root@192.168.1.1
/etc/init.d/hellodpi enable
/etc/init.d/hellodpi start
```

Servis durumunu kontrol etmek için:
```bash
logread -e hellodpi
```

---

## 2. Linux (systemd) Kurulumu

Ubuntu, Debian, Fedora, Arch Linux veya Raspberry Pi OS için:

### Adım 1: Derleme veya Binary İndirme
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o hellodpi ./cmd/hellodpi
sudo cp hellodpi /usr/local/bin/
sudo chmod +x /usr/local/bin/hellodpi
```

### Adım 2: systemd Servisini Aktif Etme
```bash
sudo cp deploy/systemd/hellodpi.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now hellodpi
```

### Adım 3: Servis Kontrolü
```bash
systemctl status hellodpi
journalctl -u hellodpi -f
```

---

## 3. Ağ Ayarları & İstemci Yönlendirme

Hello DPI router'da `192.168.1.1:18080` portunda çift modlu (HTTP CONNECT + SOCKS5) proxy olarak dinler:

- **Seçenek A (Tarayıcı / Cihaz Ayarı):** Bilgisayar veya telefonunuzun Proxy ayarlarına Router IP'sini (`192.168.1.1:18080`) SOCKS5 veya HTTP proxy olarak girin.
- **Seçenek B (Şeffaf Proxy / redsocks / iptables):** İsteğe bağlı olarak router'ın LAN arayüzünden gelen TCP 80/443 trafiğini `redsocks` veya `iptables TPROXY` ile `18080` portuna şeffaf yönlendirebilirsiniz.
