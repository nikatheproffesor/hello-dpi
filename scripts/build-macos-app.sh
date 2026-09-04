#!/usr/bin/env bash
set -e

APP_NAME="Hello DPI"
BUNDLE_DIR="$APP_NAME.app"
CONTENTS_DIR="$BUNDLE_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

echo "=== Building $APP_NAME for macOS with Logo ==="

rm -rf "$BUNDLE_DIR"
mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"

if [ -f "assets/AppIcon.icns" ]; then
    echo "Adding AppIcon.icns to bundle..."
    cp "assets/AppIcon.icns" "$RESOURCES_DIR/AppIcon.icns"
fi

echo "Compiling binary..."
go build -ldflags="-s -w" -o "$MACOS_DIR/$APP_NAME" ./cmd/hellodpi-tray
chmod +x "$MACOS_DIR/$APP_NAME"

echo "Generating Info.plist..."
cat <<EOF > "$CONTENTS_DIR/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$APP_NAME</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>com.hellodpi.app</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.1.0</string>
    <key>CFBundleVersion</key>
    <string>1.1.0</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

echo "Packaging macOS release zip..."
mkdir -p bin
rm -f "bin/HelloDPI-macOS.zip"
zip -r -q "bin/HelloDPI-macOS.zip" "$BUNDLE_DIR"

echo "✓ Successfully generated '$BUNDLE_DIR' and 'bin/HelloDPI-macOS.zip' with logo!"
echo "  Kullanıcı sadece 'Hello DPI.app' dosyasına çift tıklayarak çalıştırabilir."
