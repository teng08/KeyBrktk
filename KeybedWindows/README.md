# Keybed for Windows (preview)

Download `Keybed-Windows-compatible.exe` when you want one easy-to-share file
for 32- or 64-bit Intel/AMD PCs running Windows 10 or 11. It is a 32-bit app and
runs through Windows' built-in compatibility layer on 64-bit PCs. Download
`Keybed-Windows-x64.exe` for the native 64-bit Intel/AMD build,
or the **experimental** `Keybed-Windows-arm64.exe` for Windows 11 on ARM, from
[GitHub Releases](https://github.com/teng08/KeyBrktk/releases).

Save the EXE in a permanent folder, then double-click it. **The one EXE includes
all sounds, credits and full license notices.** No extra files, Go, .NET runtime,
installer or administrator installation is required. Use **Credits** in Sound
Studio or the tray menu to read the bundled notices.
Optional ZIP downloads contain the same standalone EXE plus credits/instructions;
choose **Extract All** before launching one. Releases 3.5 and earlier still need
`Keybed.soundbank` beside their executable; this requirement is removed in 3.6.

This preview is unsigned. Windows may show an unknown-publisher/SmartScreen
warning. Only continue if you obtained the file from this repository and trust
it; do not disable Windows security protections. Antivirus or organizational
policies can restrict apps that observe global keyboard events.

The native Sound Studio now follows the **same dark design and layout as the Mac**:
the heading, colored sound cards and selected badges, featured Skibiddy card,
scrolling three-column library, three intensity buttons, volume, mute, release
sounds, preview, reset count and a three-second typing counter. All thirteen
presets have matching descriptions and accent colors. The new recorded switches
are immediately visible. Try typing in the text field or another application.
The studio scales for high-DPI displays, responds to monitor DPI changes and
scrolls vertically on shorter screens so the bottom controls remain reachable.
Native Windows title bars, buttons and scrollbars have platform-specific details;
the app does not require a browser or WebView runtime.

Audio samples are preloaded and up to
32 sounds can overlap. Settings persist in `%APPDATA%\Keybed\settings.json`.
Holy Pandas, NovelKeys Creams and Turquoise Tealios use bundled recordings, with
dedicated Space/Enter/Backspace and release samples. Original preset indices are
preserved so settings from the earlier release still select the same sound.

Closing the studio hides it and leaves keyboard sounds running. Double-click the
notification-area icon to reopen it, or right-click the icon to choose sounds,
intensity, floating-counter visibility, reset, mute, preview or
quit. Use **Quit Keybed** to stop the app. Only one instance runs at a time.
If Explorer/the notification area is unavailable, closing the studio minimizes
it rather than hiding it, and the app retries tray registration. It never leaves
you with a hidden studio and no tray entry.

**Known ARM64 limitation:** tray registration failed on the Windows ARM64 CI
runner even though Explorer was present. The experimental ARM64 download may
use the minimize-to-taskbar fallback instead of a tray icon. Reopen the studio
from the taskbar and use **Quit Keybed** to stop it. ARM64 tray integration is
not verified; the x64 download passes a strict tray check.

The hook always forwards keyboard events unchanged. Key characters are never
saved or transmitted. Only temporary key-down state, an aggregate count and
timestamps are retained in memory. Software-injected keys are ignored.
The listener is limited to the current interactive Windows desktop; sign-in,
UAC's secure desktop and protected/elevated applications may not be observable.

Enable **Floating counter follows my mouse** for the same small dark counter as
on Mac. It shows the current sound and counts each key once per press/release
cycle; holding a key does not repeat its sound or count,
resets its typing window every three seconds, and collapses after an idle pause.
It is click-through, never takes focus and stays within the cursor's monitor work
area. Windows' animation preference disables pulsing and animated collapse.
Closing/minimizing the studio does not stop the overlay or sounds. Mute
still counts; previews and releases do not. **Reset count** clears the counter.
Overlay visibility is saved with the other settings.

**Start automatically when I log in** is off by default on Windows. Checking it
registers this exact executable in the current user's Windows Run key and starts
Keybed in the background at sign-in. Unchecking it removes that setting; no
administrator access is needed. Keep the executable in its permanent
location before enabling startup. A manually created Startup-folder shortcut
from an older version is separate and must be removed manually if not wanted.
Mac continues to use its existing login-agent and Input Monitoring integration;
Windows installs its own keyboard hook instead of showing a Mac permission button.

## Verify the download

Run in PowerShell from the download's folder (use its downloaded filename):

```powershell
.\Keybed-Windows-x64.exe --self-test
if ($LASTEXITCODE -ne 0) { throw "Keybed self-test failed" }
```

This checks all 468 sounds through the production mixer without listening to
the keyboard or requiring an audio device. It does not prove that your sound
device, security policy or global hook works. For a real-device check, open
Notepad, type, test Space/Enter/Delete, change presets, mute, hide/reopen the
studio, and switch the audio output. If an audio device stops responding,
Keybed retries the default output instead of exiting.

## Build from source

Go 1.20 or newer is required for rebuilding only. The Windows app has no external
Go dependencies and does not use cgo. `build.ps1` stages the bank and credits
into ignored `assets/`, then builds with `-tags bundled` to embed them. It produces
both the single-file EXE and the optional ZIP. Normal `go build` without this tag
still supports a sidecar bank for source development, or an explicit `--bank PATH`.

Download `Keybed.soundbank` from the same release into `dist/common/`, or generate
it on a Mac with Apple's Command Line Tools:

```sh
KEYBED_APP_OUTPUT="$PWD/dist/macos/Keybed.app" ./KeybedMac/build-app.sh
./dist/macos/Keybed.app/Contents/MacOS/Keybed \
  --export-windows-sounds "$PWD/dist/common/Keybed.soundbank"
```

The export uses the Mac production sound pipeline, including the bundled sample
recordings and speech. It runs only while building; Windows playback is offline
and does not require a Mac.

On Windows, from the repository root:

```powershell
.\KeybedWindows\build.ps1
# One-file 32-/64-bit Intel/AMD compatibility build:
.\KeybedWindows\build.ps1 -Architecture 386
# Experimental native Windows-on-ARM package:
.\KeybedWindows\build.ps1 -Architecture arm64

$env:KEYBED_SOUND_BANK = "$PWD\dist\common\Keybed.soundbank"
Push-Location KeybedWindows
go test ./...
Pop-Location
```

GitHub Actions builds and runs tests separately on Windows x86 compatibility,
x64 and ARM64.
The release EXE is tested alone in an empty folder, without its ZIP, credits
folder or a sidecar bank. Tests also compare its embedded bank and full notices
byte-for-byte with the source/export, and open/resize/close the credits viewer.
The CI smoke test clicks every sound card and intensity button, verifies scrolling
and small-screen footer access, exercises 100/150/200% DPI layout changes, and
checks floating-counter enable/disable, counting, idle collapse, focus and reset.
Native screenshots are saved as workflow artifacts for visual review. It also
checks keyboard-hook installation, close/reopen
and shutdown with audio disabled, because hosted runners may not have a sound
device. Windows x64 must also pass tray registration. The experimental ARM64
smoke test explicitly permits the documented tray limitation, reports it and
verifies the minimize/reopen fallback instead; it does not claim the tray works.
Both builds also exercise the fallback when a working tray is present.
Real-device playback still needs manual verification.
Smoke tests never change the user's login setting or saved preferences.
