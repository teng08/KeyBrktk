#!/bin/zsh
set -euo pipefail
cd "$(dirname "$0")"
# Use the native engine and controls. No Go installation or /tmp checkout needed.
exec ../KeybedMac/build-app.sh "$@"
