# Keybed for Windows (preview)

Download `Keybed-Windows-x64.zip` for Intel/AMD PCs running Windows 10 or 11,
or `Keybed-Windows-arm64.zip` for Windows 11 on ARM, from
[GitHub Releases](https://github.com/teng08/KeyBrktk/releases).

Right-click the ZIP and choose **Extract All**, then double-click `Keybed.exe`.
Keep `Keybed.soundbank` beside the executable. No Go installation, .NET runtime,
installer or administrator access is required to run the download.

This preview is unsigned. Windows may show an unknown-publisher/SmartScreen
warning. Only continue if you obtained the file from this repository and trust
it; do not disable Windows security protections. Antivirus or organizational
policies can restrict apps that observe global keyboard events.

The native Sound Studio provides the same ten sound presets and three intensity
settings as the Mac app, along with volume, mute, key release sounds, sound
preview and a three-second typing counter. Audio samples are preloaded and up to
32 sounds can overlap. Settings persist in `%APPDATA%\Keybed\settings.json`.

Closing the studio hides it and leaves keyboard sounds running. Double-click the
notification-area icon to reopen it, or right-click the icon to mute, preview or
quit. Use **Quit Keybed** to stop the app. Only one instance runs at a time.

The hook always forwards keyboard events unchanged. Key characters are never
saved or transmitted. Only temporary key-down state, an aggregate count and
timestamps are retained in memory. Software-injected keys are ignored.
The listener is limited to the current interactive Windows desktop; sign-in,
UAC's secure desktop and protected/elevated applications may not be observable.

The Windows interface is native rather than a copy of the Mac studio.
The Mac's mouse-following floating HUD and automatic login setup are not yet
included in this Windows preview. To launch at login, you can add a shortcut
to `Keybed.exe` to the user Startup folder (`shell:startup`).

## Verify the download

Run in PowerShell from the extracted folder:

```powershell
.\Keybed.exe --self-test
if ($LASTEXITCODE -ne 0) { throw "Keybed self-test failed" }
```

This checks all 360 sounds through the production mixer without listening to
the keyboard or requiring an audio device. It does not prove that your sound
device, security policy or global hook works. For a real-device check, open
Notepad, type, test Space/Enter/Delete, change presets, mute, hide/reopen the
studio, and switch the audio output. If an audio device stops responding,
Keybed retries the default output instead of exiting.

## Build from source

Go 1.20 or newer is required for rebuilding only. The Windows app has no external
Go dependencies and does not use cgo.

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
# Native Windows-on-ARM package:
.\KeybedWindows\build.ps1 -Architecture arm64

$env:KEYBED_SOUND_BANK = "$PWD\dist\common\Keybed.soundbank"
Push-Location KeybedWindows
go test ./...
Pop-Location
```

GitHub Actions builds and runs tests separately on Windows x64 and Windows ARM64.
The CI smoke test checks native window/control/tray creation, keyboard-hook
installation and shutdown with audio disabled, because hosted runners may not
have a sound device. Real-device playback still needs manual verification.
