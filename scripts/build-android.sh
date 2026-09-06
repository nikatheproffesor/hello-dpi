#!/usr/bin/env bash
set -e

echo "=== Building Hello DPI Android Engine ==="

mkdir -p bin
mkdir -p mobile/android/app/src/main/jniLibs/arm64-v8a

echo "Compiling Android ARM64 native binary..."
GOOS=android GOARCH=arm64 go build -ldflags="-s -w" -o mobile/android/app/src/main/jniLibs/arm64-v8a/libhellodpi.so ./cmd/hellodpi
cp mobile/android/app/src/main/jniLibs/arm64-v8a/libhellodpi.so bin/hellodpi-android-arm64

echo "Android native binary ready at: bin/hellodpi-android-arm64 and mobile/android/app/src/main/jniLibs/arm64-v8a/libhellodpi.so"

echo "Building Android APK via Gradle..."
(
    cd mobile/android
    ./gradlew assembleRelease
    cp app/build/outputs/apk/release/app-release.apk ../../bin/HelloDPI-Android.apk
)

echo "✓ Android build complete! APK ready at: bin/HelloDPI-Android.apk"
