#!/usr/bin/env bash
set -e

APP_NAME="Hello DPI"
BUNDLE_DIR="$APP_NAME.app"
CONTENTS_DIR="$BUNDLE_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

echo "=== Building $APP_NAME for macOS (Ad-Hoc Signed) ==="

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
    <string>4.0.0</string>
    <key>CFBundleVersion</key>
    <string>4.0.0</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

IDENTITY=$(security find-identity -v -p codesigning 2>/dev/null | grep "Developer ID Application" | head -n 1 | awk '{print $2}' || true)
if [ -z "$IDENTITY" ]; then
    IDENTITY=$(security find-identity -v -p codesigning 2>/dev/null | grep "Apple Development" | head -n 1 | awk '{print $2}' || true)
fi

if [ -n "$IDENTITY" ]; then
    echo "Applying signature with Keychain Identity: $IDENTITY..."
    codesign --force --deep --options runtime --timestamp --sign "$IDENTITY" "$BUNDLE_DIR"
else
    echo "Applying Ad-Hoc Code Signature..."
    codesign --force --deep --sign - "$BUNDLE_DIR"
fi

# Clear quarantine if any
xattr -cr "$BUNDLE_DIR" 2>/dev/null || true

mkdir -p bin

echo "Creating macOS release ZIP..."
rm -f "bin/HelloDPI-macOS.zip"
zip -r -q "bin/HelloDPI-macOS.zip" "$BUNDLE_DIR"

echo "Creating macOS release DMG..."
rm -f "bin/HelloDPI-macOS.dmg"
DMG_TMP="dmg_tmp"
rm -rf "$DMG_TMP"
mkdir -p "$DMG_TMP"
cp -R "$BUNDLE_DIR" "$DMG_TMP/"
ln -s /Applications "$DMG_TMP/Applications"
hdiutil create -volname "Hello DPI" -srcfolder "$DMG_TMP" -ov -format UDZO "bin/HelloDPI-macOS.dmg" -quiet
rm -rf "$DMG_TMP"

echo "✓ Successfully built and signed '$BUNDLE_DIR', DMG and ZIP!"
