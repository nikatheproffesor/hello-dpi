#!/usr/bin/env bash
set -e

echo "=== Hello DPI Linux Installer ==="

INSTALL_DIR="$HOME/.local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_FILE="$SERVICE_DIR/hellodpi.service"
BINARY_SOURCE="./bin/hellodpi-linux-amd64"

if [ ! -f "$BINARY_SOURCE" ]; then
    BINARY_SOURCE="./bin/hellodpi"
fi

if [ ! -f "$BINARY_SOURCE" ]; then
    echo "Building Hello DPI binary..."
    go build -o bin/hellodpi ./cmd/hellodpi
    BINARY_SOURCE="./bin/hellodpi"
fi

mkdir -p "$INSTALL_DIR"
cp "$BINARY_SOURCE" "$INSTALL_DIR/hellodpi"
chmod +x "$INSTALL_DIR/hellodpi"

mkdir -p "$SERVICE_DIR"
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Hello DPI Zero-Overhead Anti-Censorship Service
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/hellodpi -addr 127.0.0.1:8080 -system-proxy
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF

systemctl --user daemon-reload
systemctl --user enable hellodpi.service
systemctl --user restart hellodpi.service

echo "✓ Hello DPI successfully installed and running in the background as systemd user service!"
echo "  • Check status: systemctl --user status hellodpi"
echo "  • View logs:    journalctl --user -u hellodpi -f"
