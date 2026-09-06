#!/usr/bin/env bash
set -e

echo "=== Building Universal HelloCore.xcframework for iOS & Simulators ==="

TARGET_DIR="mobile/ios/Frameworks"
mkdir -p "$TARGET_DIR"

TMP_DIR="tmp_ios_build"
rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR/device"
mkdir -p "$TMP_DIR/sim"
mkdir -p "$TMP_DIR/include"

SDK_DEVICE_PATH=$(xcrun --sdk iphoneos --show-sdk-path)
CLANG_DEVICE=$(xcrun --sdk iphoneos -f clang)

SDK_SIM_PATH=$(xcrun --sdk iphonesimulator --show-sdk-path)
CLANG_SIM=$(xcrun --sdk iphonesimulator -f clang)

echo "1. Compiling for iOS Device (arm64)..."
CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
CC="$CLANG_DEVICE -isysroot $SDK_DEVICE_PATH -arch arm64 -miphoneos-version-min=15.0" \
go build -buildmode=c-archive -o "$TMP_DIR/device/libHelloCore.a" ./pkg/hellocore/mobile

echo "2. Compiling for iOS Simulator (arm64 + x86_64)..."
CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
CC="$CLANG_SIM -isysroot $SDK_SIM_PATH -arch arm64 -mios-simulator-version-min=15.0" \
go build -buildmode=c-archive -o "$TMP_DIR/sim/libHelloCore_arm64.a" ./pkg/hellocore/mobile

CGO_ENABLED=1 GOOS=ios GOARCH=amd64 \
CC="$CLANG_SIM -isysroot $SDK_SIM_PATH -arch x86_64 -mios-simulator-version-min=15.0" \
go build -buildmode=c-archive -o "$TMP_DIR/sim/libHelloCore_x86_64.a" ./pkg/hellocore/mobile

lipo -create "$TMP_DIR/sim/libHelloCore_arm64.a" "$TMP_DIR/sim/libHelloCore_x86_64.a" -output "$TMP_DIR/sim/libHelloCore.a"

echo "3. Preparing C Headers & Module Map..."
cp "$TMP_DIR/device/libHelloCore.h" "$TMP_DIR/include/HelloCore.h"

cat <<MAP > "$TMP_DIR/include/module.modulemap"
module HelloCore {
    header "HelloCore.h"
    export *
}
MAP

echo "4. Generating universal XCFramework..."
rm -rf "$TARGET_DIR/HelloCore.xcframework"
xcodebuild -create-xcframework \
  -library "$TMP_DIR/device/libHelloCore.a" \
  -headers "$TMP_DIR/include" \
  -library "$TMP_DIR/sim/libHelloCore.a" \
  -headers "$TMP_DIR/include" \
  -output "$TARGET_DIR/HelloCore.xcframework"

rm -rf "$TMP_DIR"
echo "✓ Successfully generated $TARGET_DIR/HelloCore.xcframework!"
