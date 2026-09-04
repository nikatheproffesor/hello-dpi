#!/usr/bin/env bash
set -e

echo "=== Hello DPI macOS Installer ==="

INSTALL_DIR="$HOME/.local/bin"
PLIST_PATH="$HOME/Library/LaunchAgents/com.hellodpi.service.plist"
BINARY_SOURCE="./bin/hellodpi"

if [ ! -f "$BINARY_SOURCE" ]; then
    echo "Building Hello DPI binary..."
    go build -o bin/hellodpi ./cmd/hellodpi
fi

mkdir -p "$INSTALL_DIR"
cp "$BINARY_SOURCE" "$INSTALL_DIR/hellodpi"
chmod +x "$INSTALL_DIR/hellodpi"

echo "Configuring launchd background service..."
mkdir -p "$HOME/Library/LaunchAgents"

cat <<EOF > "$PLIST_PATH"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.hellodpi.service</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/hellodpi</string>
        <string>-addr</string>
        <string>127.0.0.1:8080</string>
        <string>-system-proxy</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/hellodpi.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/hellodpi.err</string>
</dict>
</plist>
EOF

launchctl unload "$PLIST_PATH" 2>/dev/null || true
launchctl load "$PLIST_PATH"

echo "✓ Hello DPI successfully installed and running in the background!"
echo "  • Status: Active on 127.0.0.1:8080"
echo "  • System proxy automatically enabled."
echo "  • Logs: tail -f /tmp/hellodpi.log"
