#!/bin/zsh
set -euo pipefail
cd "$(dirname "$0")"
PROJECT_DIR="$(cd .. && pwd)"
SOURCE_APP="$PROJECT_DIR/Keybed.app"
TARGET_APP="/Applications/Keybed.app"
if [[ ! -x "$SOURCE_APP/Contents/MacOS/Keybed" ]]; then
    ./build-app.sh
fi
if [[ -e "$TARGET_APP" ]]; then
    EXISTING_ID="$(plutil -extract CFBundleIdentifier raw "$TARGET_APP/Contents/Info.plist")"
    if [[ "$EXISTING_ID" != com.keybed.keyboard-sound ]]; then
        echo "Another application is named Keybed.app in Applications. It was not replaced." >&2
        exit 1
    fi
fi
INSTALL_DIR="$(mktemp -d /Applications/.keybed-install.XXXXXX)"
trap 'rm -rf "$INSTALL_DIR"' EXIT
ditto "$SOURCE_APP" "$INSTALL_DIR/Keybed.app"
codesign --verify --strict "$INSTALL_DIR/Keybed.app"
"$INSTALL_DIR/Keybed.app/Contents/MacOS/Keybed" --self-test
"$SOURCE_APP/Contents/MacOS/Keybed" --stop-running
if [[ -e "$TARGET_APP" ]]; then
    mv "$TARGET_APP" "$INSTALL_DIR/previous-Keybed.app"
fi
if ! mv "$INSTALL_DIR/Keybed.app" "$TARGET_APP"; then
    if [[ -e "$INSTALL_DIR/previous-Keybed.app" ]]; then
        mv "$INSTALL_DIR/previous-Keybed.app" "$TARGET_APP"
    fi
    exit 1
fi
if [[ "${1:-}" != --no-login ]]; then
    "$TARGET_APP/Contents/MacOS/Keybed" --enable-login
fi
if [[ -n "${KEYBED_DIAGNOSTIC_OUTPUT:-}" ]]; then
    open "$TARGET_APP" --args --diagnostic-output "$KEYBED_DIAGNOSTIC_OUTPUT"
else
    open "$TARGET_APP"
fi
echo "Installed $TARGET_APP"
