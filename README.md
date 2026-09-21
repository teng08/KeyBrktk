# Keybed

## Downloads and platform support

Prebuilt preview downloads are published in
[GitHub Releases](https://github.com/teng08/KeyBrktk/releases) after platform checks pass.
**Download the app below, not GitHub's "Source code" archives. No coding or building is needed.**

| Device | Download | Minimum system |
| --- | --- | --- |
| Intel Mac or Apple-silicon MacBook/Mac | [Download Mac app (.dmg)](https://github.com/teng08/KeyBrktk/releases/download/v3.6.1-preview.1/Keybed-macOS-universal.dmg) | macOS 11 |
| Most Windows PCs (easy-to-share compatibility build) | [Download compatible Windows app (.exe)](https://github.com/teng08/KeyBrktk/releases/download/v3.6.1-preview.1/Keybed-Windows-compatible.exe) | Windows 10/11, 32- or 64-bit Intel/AMD |
| Intel/AMD Windows PC | [Download native 64-bit Windows app (.exe)](https://github.com/teng08/KeyBrktk/releases/download/v3.6.1-preview.1/Keybed-Windows-x64.exe) | Windows 10/11 x64 |
| Windows-on-ARM PC (experimental) | [Download Windows ARM app (.exe)](https://github.com/teng08/KeyBrktk/releases/download/v3.6.1-preview.1/Keybed-Windows-arm64.exe) | Windows 11 |

The Mac binary contains both `x86_64` and `arm64`, so Apple-silicon MacBooks run
natively without Rosetta. **Mac:** open the DMG, drag Keybed onto Applications,
eject the disk image, then open Keybed from Applications. Grant **Input Monitoring**
and quit/reopen when macOS asks. If updating, quit the old app before replacing it.
**Windows:** save the EXE in a permanent folder and double-click it. Sounds and
full license notices are built into this single file; no sound-bank download,
extraction, installer, Go, .NET, or administrator installation is required.
The `compatible` download is a 32-bit executable that also runs on normal 64-bit
Intel/AMD Windows, making it the simplest single file to share. The native `x64`
download remains preferable when the recipient's PC type is known. Windows on
ARM should use the experimental `arm64` download.
ZIP downloads remain available as alternatives. Mac ZIPs contain `Keybed.app`;
Windows ZIPs contain the same standalone EXE plus readable credits/instructions.
Older Windows releases (3.5 and earlier) still require their sidecar sound bank.
See [Windows instructions](KeybedWindows/README.md) for details.

Both desktop apps share the **dark Sound Studio design**: matching colored sound
cards, descriptions, selected badges, scrolling sound library, intensity, volume,
release sounds, mute/preview, reset count and a mouse-following floating counter.
Intel and Apple-silicon Macs use the same universal Mac app; Windows x64 and
ARM64 use the matching native Windows studio. Windows also has the same startup
checkbox, off by default, and a DPI-aware/short-screen layout. OS title bars and
controls retain native differences. Mac needs Input Monitoring; Windows uses its
keyboard hook and notification area rather than the Mac permission button/menu bar.

The native Windows ARM64 build is **experimental**: tray registration failed on
its CI runner even with Explorer present. If the tray is unavailable, Close
minimizes the studio instead of hiding it; reopen it from the taskbar. CI checks
that fallback and the native controls/hook, but does **not** mark ARM64 tray
integration as verified. Windows x64 still requires its tray check to pass.

Downloads are not commercially signed/notarized yet: the Mac bundle is ad-hoc
signed and Windows executables are unsigned. OS security warnings are possible.
Only run downloads you trust, and do not disable system security protections.
Developer ID signing/notarization and Windows code-signing certificates are
needed for a smoother public installation experience.

GitHub Actions builds and tests Intel and Apple-silicon Macs, Windows compatible
(32-bit x86), x64 and ARM64. Automated sound/ABI/UI-hook smoke checks still need manual
playback testing on real devices; CI does not prove every audio device or security
policy works. The browser demo is separate and plays sounds only while its page
is focused; it is not a replacement for either global desktop listener.
Workflow artifacts include native Studio screenshots from all five build targets.

## Mac integration

For Mac integration, install Keybed into Applications:

For the prebuilt download, use the DMG instructions above. The command below is
for installing a **local source build**, not required for people downloading the app:

```sh
./KeybedMac/install-app.sh
```

This installs **/Applications/Keybed.app** and enables startup at login. Keybed
runs as a menu bar app without a Dock icon, continues after closing its window,
and avoids App Nap while listening. Uncheck **Start automatically when I log in**
in Keybed to disable automatic startup. Use `install-app.sh --no-login` to install
without enabling login startup. The login agent is
`~/Library/LaunchAgents/com.keybed.keyboard-sound.login.plist`; it opens the
installed app in the background at sign-in.

The red **×** button and **⌘W** hide the Sound Studio and keep Keybed's audio,
keyboard listener, floating overlay and timers running in the menu bar. Reopen
the studio using the keyboard icon → **Show Keybed**. **Quit Keybed / ⌘Q** stops
the app and its sounds. Keybed also retries its audio engine if it stops while
running in the background.

The first launch requests **Keybed's own Input Monitoring permission**. Allow
the installed app in the Privacy panel, then quit and reopen it if macOS asks.
An open event tap alone is not treated as proof of global keyboard access.
Keybed shows **Keyboard access needed** until global permission is granted.
If a previous Keybed entry is already enabled but sounds only work in its own
window, remove that entry and use **+** to add **/Applications/Keybed.app**.
Accessibility access, if already granted, also provides a global event fallback.
Secure password entry can temporarily prevent keyboard observation.

If sounds stop after hiding the studio, check the keyboard icon's status. An
orange icon with **Keyboard access needed** means macOS allows sounds only in
Keybed's own window. **Enable keyboard access…** opens Input Monitoring and
reveals the exact app copy in Finder. On Monterey, unlock the padlock if needed,
use **+** to add `/Applications/Keybed.app`, then check Keybed. Remove an obsolete
Keybed entry first if it is checked but the installed app still reports no access.
Accept macOS's quit/reopen request. A locally signed rebuild can require fresh
permission; always enable the final installed copy.

Double-click **Keybed.app** in this folder. The app is built for this Mac and
contains its sound recordings; running it does not require Go or a terminal.

On first launch, click **Enable keyboard access**, then enable **Keybed** in
**System Settings → Privacy & Security → Input Monitoring**. On macOS Monterey,
this is **System Preferences → Security & Privacy → Privacy → Input Monitoring**.
If macOS asks you to quit and reopen, reopen Keybed.app. If Keybed is missing from
the list, use the **+** button to add this app. **Test sound** works before granting
keyboard access.

The **Sound Studio** has thirteen presets: **Alpaca Linear, Tactile Brown, Clicky Blue,
Deep Thock, Creamy Marble, Typewriter, Bubble Pop, Pixel Tap, Birdy Chirp, Skibiddy Toilet,
Holy Pandas, NovelKeys Creams and Turquoise Tealios**.
Click a sound card to select and preview it, or choose a sound from the menu bar.
New presets have a **NEW** badge. Try the text field to hear actual typing.
The six mechanical presets use bundled recordings; six presets are
synthesized at startup. **Skibiddy Toilet** uses generated robot vocals saying
“skibiddy”, “toilet”, “dop dop” and “yes yes”. Enter plays “skibiddy toilet”.
All sounds work offline and are preloaded; speech is never generated on a key press.
The three new switch presets use recorded samples, including distinct Space,
Enter, Delete and release sounds, not renamed synthesized profiles. Scroll the
Mac sound library to reach all cards. Existing saved selections stay unchanged.

The animated floating counter follows your **mouse pointer**, displays the
current sound and counts each physical key press once. Holding a key does not
retrigger its sound or increase the count until that key is released. It is
click-through and never takes keyboard focus. The count resets every **3 seconds** from the first key of each typing window,
even during continuous typing. After **3 seconds without typing**, the counter
collapses to a small badge showing your sound; typing expands it again.
**Reset count** clears it immediately. Previews and key releases do not count,
and muted typing still counts. Disable **Floating counter follows my mouse** to
hide it. It stays within the current display and respects Reduce Motion.

**Aggressive** is the default. Choose **Extreme** for a brighter, harder hit, or
**Balanced** for a softer sound. Volume and key release sounds are adjustable;
these settings, your selected sound and overlay visibility persist when reopening
the app. Space, Enter and Delete have their own sound variants. Closing the window
leaves Keybed running; use the keyboard
icon in the menu bar to reopen, mute or quit it.

The native audio engine stays running and mixes up to 32 overlapping sounds from
memory. All 468 preset/intensity/key combinations are preloaded. MP3 decoding,
synthesis, silence trimming, normalization and tone processing happen at startup.
A dedicated keyboard event thread feeds the audio callback, which
does not allocate memory or wait for a mutex. Playback starts in the next audio
block, and Windows uses small interactive output buffers; actual audible latency
depends on the output device. Bluetooth audio adds
its own latency. Keybed does not save or transmit what you type.

## Rebuild

Apple's Command Line Tools are required only to rebuild:

```sh
./KeybedGo/build-app.sh
# Or directly:
./KeybedMac/build-app.sh
```

Both build scripts create the same native app from local files, verify its bundled
audio, and sign it locally before replacing the previous build. No `/tmp` checkout
or external Go module is required. The script builds a universal Intel/Apple-silicon
binary for macOS 11 or newer. Set `KEYBED_APP_OUTPUT` to package elsewhere without
replacing the local app.

`./start-keybed.sh` opens the installed app if present, or the local build,
building only if it is missing.
`./start-keybed.sh --build` rebuilds before opening.

The optional Go launcher remains available:

```sh
cd KeybedGo
go run .
go run . --build
go run . --self-test
```

The launcher opens the installed Applications copy when present. `--build`
rebuilds the local app; rerun the installer to update the Applications copy.

## Checks

```sh
./Keybed.app/Contents/MacOS/Keybed --self-test
./KeybedMac/test.sh
./Keybed.app/Contents/MacOS/Keybed --smoke-test
```

Sound credits and the recording license are bundled with the app and available
in [Sounds/CREDITS.md](Sounds/CREDITS.md).

## Release builds

[Desktop builds](.github/workflows/desktop.yml) runs for pushes to `main`, pull
requests and manual workflow dispatches. Successful runs include downloadable
app ZIP artifacts. Pushing a new `v*` tag also publishes a preview GitHub Release,
but only after every platform check succeeds; it never overwrites an existing
release. Artifacts include the universal Mac DMG/ZIP, single-file Windows
compatible/x64/ARM64 EXEs and optional ZIPs, a developer-only sound bank, and
SHA-256 checksums.
The Mac packaging step mounts the finished DMG read-only and verifies both
architectures, signatures, sounds and bundled license. Windows tests copy just
the EXE into an otherwise empty folder before sound and native UI smoke checks.

The Windows sound bank is generated by the Mac production pipeline during the
build, so every desktop uses the same 468 prepared sound samples. Windows
playback itself is fully offline. See [Windows rebuilding](KeybedWindows/README.md#build-from-source).

Checks also verify the recording hashes/licenses, identical Mac/Windows preset
catalogs, saved-selection compatibility and every preset's control-to-mixer
mapping. Mac CI exercises scrolling, all preset selections, the HUD/counter and
close/reopen with audio output intentionally disabled. A local Mac smoke test
without `--no-audio` additionally checks that the audio engine stays running.
Windows smoke tests likewise do not require an audio device. Physical playback
and OS keyboard permissions still require a real-device check.
