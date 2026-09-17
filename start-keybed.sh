#!/bin/zsh
set -euo pipefail
cd "$(dirname "$0")"
if [[ "${1:-}" != "--build" && -x /Applications/Keybed.app/Contents/MacOS/Keybed ]]; then
    exec /usr/bin/open /Applications/Keybed.app
fi
if [[ "${1:-}" == "--build" || ! -x Keybed.app/Contents/MacOS/Keybed ]]; then
    ./KeybedMac/build-app.sh
fi
exec /usr/bin/open ./Keybed.app
