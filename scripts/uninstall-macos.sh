#!/usr/bin/env bash
set -e

echo "=== Hello DPI macOS Uninstaller ==="

PLIST_PATH="$HOME/Library/LaunchAgents/com.hellodpi.service.plist"
INSTALL_DIR="$HOME/.local/bin"

if [ -f "$PLIST_PATH" ]; then
    echo "Stopping and unloading launchd service..."
    launchctl unload "$PLIST_PATH" 2>/dev/null || true
    rm -f "$PLIST_PATH"
fi

rm -f "$INSTALL_DIR/hellodpi"

echo "Clearing system proxy settings..."
for service in $(networksetup -listallnetworkservices | grep -v '*' || true); do
    networksetup -setwebproxystate "$service" off 2>/dev/null || true
    networksetup -setsecurewebproxystate "$service" off 2>/dev/null || true
    networksetup -setsocksfirewallproxystate "$service" off 2>/dev/null || true
done

echo "✓ Hello DPI completely uninstalled and proxy settings restored."
