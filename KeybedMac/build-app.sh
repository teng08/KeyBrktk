#!/bin/zsh
set -euo pipefail
cd "$(dirname "$0")"
PROJECT_DIR="$(cd .. && pwd)"
APP_DIR="${KEYBED_APP_OUTPUT:-$PROJECT_DIR/Keybed.app}"
mkdir -p "$(dirname "$APP_DIR")"
# Build and validate a new bundle before replacing the working app.
BUILD_DIR="$(mktemp -d "$PROJECT_DIR/.keybed-build.XXXXXX")"
trap 'if [[ "${KEYBED_KEEP_BUILD:-0}" != 1 ]]; then rm -rf "$BUILD_DIR"; fi' EXIT
STAGED_APP="$BUILD_DIR/Keybed.app"
mkdir -p "$STAGED_APP/Contents/MacOS" "$STAGED_APP/Contents/Resources/Sounds"
MODULE_CACHE="$PROJECT_DIR/KeybedMac/.build/module-cache"
mkdir -p "$MODULE_CACHE"
export CLANG_MODULE_CACHE_PATH="$MODULE_CACHE"
export SWIFT_MODULECACHE_PATH="$MODULE_CACHE"
if ! xcrun --find swiftc >/dev/null 2>&1; then
    echo "Install Apple's Command Line Tools with: xcode-select --install" >&2
    exit 1
fi
SDK_PATH="$(xcrun --sdk macosx --show-sdk-path)"
# One download runs natively on Intel and Apple silicon, without Rosetta.
for CPU_ARCH in x86_64 arm64; do
    xcrun clang -O3 -std=c11 -arch "$CPU_ARCH" -isysroot "$SDK_PATH" -mmacosx-version-min=11.0 \
        -I Sources/AudioMixer/include -c Sources/AudioMixer/AudioMixer.c -o "$BUILD_DIR/AudioMixer-$CPU_ARCH.o"
    xcrun swiftc -O -module-cache-path "$MODULE_CACHE" -sdk "$SDK_PATH" \
        -target "$CPU_ARCH-apple-macosx11.0" -I Sources/AudioMixer/include \
        Sources/KeybedMac/*.swift "$BUILD_DIR/AudioMixer-$CPU_ARCH.o" \
        -o "$BUILD_DIR/Keybed-$CPU_ARCH" \
        -framework AppKit -framework AVFoundation -framework ApplicationServices
done
xcrun lipo -create "$BUILD_DIR/Keybed-x86_64" "$BUILD_DIR/Keybed-arm64" \
    -output "$STAGED_APP/Contents/MacOS/Keybed"
xcrun lipo "$STAGED_APP/Contents/MacOS/Keybed" -verify_arch x86_64 arm64
cp -R "$PROJECT_DIR/Sounds/Alpaca" "$STAGED_APP/Contents/Resources/Sounds/Alpaca"
for SOUND_DIRECTORY in HolyPanda NovelKeysCreams TurquoiseTealios Mechanical; do
    cp -R "$PROJECT_DIR/Sounds/$SOUND_DIRECTORY" "$STAGED_APP/Contents/Resources/Sounds/$SOUND_DIRECTORY"
done
mkdir -p "$STAGED_APP/Contents/Resources/Sounds/Tactile"
cp "$PROJECT_DIR"/Sounds/Tactile/stav-tactile-*.mp3 "$STAGED_APP/Contents/Resources/Sounds/Tactile/"
cp -R "$PROJECT_DIR/Sounds/BlueSwitch" "$STAGED_APP/Contents/Resources/Sounds/BlueSwitch"
mkdir -p "$STAGED_APP/Contents/Resources/Sounds/Skibiddy"
cp "$PROJECT_DIR"/Sounds/Skibiddy/*.wav "$STAGED_APP/Contents/Resources/Sounds/Skibiddy/"
cp "$PROJECT_DIR/Sounds/Skibiddy/README.md" "$STAGED_APP/Contents/Resources/Sounds/Skibiddy/"
cp "$PROJECT_DIR/Sounds/CREDITS.md" "$STAGED_APP/Contents/Resources/Sounds/CREDITS.md"
"$STAGED_APP/Contents/MacOS/Keybed" --make-icon "$BUILD_DIR/Keybed.iconset"
iconutil -c icns "$BUILD_DIR/Keybed.iconset" -o "$STAGED_APP/Contents/Resources/Keybed.icns"
cat > "$STAGED_APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleDisplayName</key><string>Keybed</string>
<key>CFBundleExecutable</key><string>Keybed</string>
<key>CFBundleIdentifier</key><string>com.keybed.keyboard-sound</string>
<key>CFBundleName</key><string>Keybed</string>
<key>CFBundleIconFile</key><string>Keybed</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>3.6</string>
<key>CFBundleVersion</key><string>10</string>
<key>LSUIElement</key><true/>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>NSHighResolutionCapable</key><true/>
<key>NSInputMonitoringUsageDescription</key><string>Keybed plays mechanical sounds when you press and release keys in other apps.</string>
</dict></plist>
PLIST
plutil -lint "$STAGED_APP/Contents/Info.plist"
codesign --force --sign - --timestamp=none "$STAGED_APP"
"$STAGED_APP/Contents/MacOS/Keybed" --self-test
codesign --verify --strict "$STAGED_APP"
if [[ -e "$APP_DIR" ]]; then
    mv "$APP_DIR" "$BUILD_DIR/previous-Keybed.app"
fi
if ! mv "$STAGED_APP" "$APP_DIR"; then
    if [[ -e "$BUILD_DIR/previous-Keybed.app" ]]; then
        mv "$BUILD_DIR/previous-Keybed.app" "$APP_DIR"
    fi
    exit 1
fi
echo "Built $APP_DIR — double-click Keybed.app to open."
