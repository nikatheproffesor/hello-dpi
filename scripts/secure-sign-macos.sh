#!/usr/bin/env bash
set -e

APP_NAME="Hello DPI"
BUNDLE_DIR="$APP_NAME.app"
DMG_PATH="bin/HelloDPI-macOS.dmg"

echo "========================================================="
echo "   Hello DPI Secure macOS Signing & Notarization       "
echo "========================================================="

# 1. Detect signing identity from Keychain
IDENTITY=$(security find-identity -v -p codesigning | grep "Developer ID Application" | head -n 1 | awk -F '"' '{print $2}' || true)

if [ -z "$IDENTITY" ]; then
    IDENTITY=$(security find-identity -v -p codesigning | grep "Apple Development" | head -n 1 | awk -F '"' '{print $2}' || true)
fi

if [ -n "$IDENTITY" ]; then
    echo "✓ Found Keychain Signing Identity: '$IDENTITY'"
else
    echo "⚠️ No Developer ID found in Keychain. Using local ad-hoc signature (-)."
    IDENTITY="-"
fi

# 2. Build application if not already built
./scripts/build-macos-app.sh

# 3. Apply hardened runtime signature
echo "Signing '$BUNDLE_DIR' with hardened runtime..."
if [ "$IDENTITY" != "-" ]; then
    codesign --force --deep --options runtime --timestamp --sign "$IDENTITY" "$BUNDLE_DIR"
else
    codesign --force --deep --sign - "$BUNDLE_DIR"
fi

# 4. Re-package signed DMG
echo "Re-creating signed DMG package..."
DMG_TMP="dmg_tmp"
rm -rf "$DMG_TMP" "$DMG_PATH"
mkdir -p "$DMG_TMP"
cp -R "$BUNDLE_DIR" "$DMG_TMP/"
ln -s /Applications "$DMG_TMP/Applications"
hdiutil create -volname "Hello DPI" -srcfolder "$DMG_TMP" -ov -format UDZO "$DMG_PATH" -quiet
rm -rf "$DMG_TMP"

if [ "$IDENTITY" != "-" ]; then
    if ! codesign --force --sign "$IDENTITY" --timestamp "$DMG_PATH"; then
        echo "⚠️ Warning: Failed to sign DMG package."
        if [ "${STRICT_SIGN:-0}" = "1" ]; then
            exit 1
        fi
    fi
fi

# 5. Check if Notarytool Keychain profile exists
if xcrun notarytool history --keychain-profile "hellodpi-profile" &>/dev/null; then
    echo "Submitting DMG to Apple Notary Service (notarytool)..."
    if xcrun notarytool submit "$DMG_PATH" --keychain-profile "hellodpi-profile" --wait; then
        echo "Stapling notarization ticket to App and DMG..."
        xcrun stapler staple "$BUNDLE_DIR"
        xcrun stapler staple "$DMG_PATH"
        echo "✓ DMG and App successfully notarized and stapled by Apple!"
    else
        echo "❌ Notarization submission failed."
        if [ "${STRICT_SIGN:-0}" = "1" ]; then
            exit 1
        fi
    fi
else
    echo "⚠️ Apple Notary Keychain profile ('hellodpi-profile') not found."
    echo "  To enable official Apple Notarization, run:"
    echo "  xcrun notarytool store-credentials \"hellodpi-profile\" --apple-id \"<apple-id>\" --team-id \"KBGS669D97\""
    if [ "${STRICT_SIGN:-0}" = "1" ]; then
        echo "❌ Failing release due to missing notarization credentials in STRICT mode."
        exit 1
    fi
fi

echo "Creating signed macOS release ZIP..."
rm -f "bin/HelloDPI-macOS.zip"
zip -r -q "bin/HelloDPI-macOS.zip" "$BUNDLE_DIR"

echo ""
echo "✓ Done! Signed artifacts ready in 'bin/'"
