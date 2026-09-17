#!/bin/zsh
set -euo pipefail
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APP_DIR="${KEYBED_APP_OUTPUT:-$PROJECT_DIR/Keybed.app}"
if [[ ! -d "$APP_DIR" ]]; then
    echo "Build Keybed.app first with ./KeybedMac/build-app.sh" >&2
    exit 1
fi
mkdir -p "$PROJECT_DIR/dist"
PACKAGE_DIR="$(mktemp -d "$PROJECT_DIR/dist/.keybed-package.XXXXXX")"
MOUNT_DIR="$PACKAGE_DIR/mounted"
MOUNTED=0
cleanup() {
    if [[ "$MOUNTED" == 1 ]]; then
        # Do not remove staging while its read-only volume is still mounted.
        hdiutil detach "$MOUNT_DIR" || return
    fi
    rm -rf "$PACKAGE_DIR"
}
trap cleanup EXIT
mkdir -p "$PACKAGE_DIR/files" "$MOUNT_DIR"
codesign --verify --strict --all-architectures "$APP_DIR"
xcrun lipo "$APP_DIR/Contents/MacOS/Keybed" -verify_arch x86_64 arm64
ditto "$APP_DIR" "$PACKAGE_DIR/files/Keybed.app"
ln -s /Applications "$PACKAGE_DIR/files/Applications"
cp "$PROJECT_DIR/KeybedMac/INSTALL.txt" "$PACKAGE_DIR/files/READ ME FIRST.txt"
STAGED_DMG="$PACKAGE_DIR/Keybed-macOS-universal.dmg"
hdiutil create -fs HFS+ -format UDZO -volname Keybed \
    -srcfolder "$PACKAGE_DIR/files" "$STAGED_DMG"
hdiutil verify "$STAGED_DMG"
hdiutil attach -readonly -nobrowse -noautoopen -mountpoint "$MOUNT_DIR" "$STAGED_DMG"
MOUNTED=1
[[ "$(readlink "$MOUNT_DIR/Applications")" == /Applications ]]
codesign --verify --strict --all-architectures "$MOUNT_DIR/Keybed.app"
xcrun lipo "$MOUNT_DIR/Keybed.app/Contents/MacOS/Keybed" -verify_arch x86_64 arm64
"$MOUNT_DIR/Keybed.app/Contents/MacOS/Keybed" --self-test
cmp "$PROJECT_DIR/Sounds/Mechanical/LICENSE" "$MOUNT_DIR/Keybed.app/Contents/Resources/Sounds/Mechanical/LICENSE"
hdiutil detach "$MOUNT_DIR"
MOUNTED=0
mv "$STAGED_DMG" "$PROJECT_DIR/dist/Keybed-macOS-universal.dmg"
ditto -c -k --sequesterRsrc --keepParent "$APP_DIR" "$PROJECT_DIR/dist/Keybed-macOS-universal.zip"
echo "Built dist/Keybed-macOS-universal.dmg and .zip. Open the DMG and drag Keybed into Applications."
