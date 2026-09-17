#!/bin/zsh
set -euo pipefail
cd "$(dirname "$0")"
TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/keybed-tests.XXXXXX")"
trap 'rm -rf "$TEST_DIR"' EXIT
xcrun clang -O2 -std=c11 -Wall -Wextra -Werror \
    -fsanitize=address,undefined -I Sources/AudioMixer/include \
    Sources/AudioMixer/AudioMixer.c Tests/AudioMixerTests.c -o "$TEST_DIR/mixer-tests"
"$TEST_DIR/mixer-tests"
../Keybed.app/Contents/MacOS/Keybed --self-test
