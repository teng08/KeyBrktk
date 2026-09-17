## Downloads

- **MacBook / Mac (Intel or Apple silicon):** `Keybed-macOS-universal.zip`.
  Extract it and drag `Keybed.app` into Applications. Requires macOS 11 or newer.
- **Windows 10/11, Intel or AMD:** `Keybed-Windows-x64.zip`.
- **Windows 11 on ARM (experimental):** `Keybed-Windows-arm64.zip`.
  Extract the complete Windows ZIP and open `Keybed.exe`, keeping the sound bank
  beside it. No Go, .NET or administrator installation is needed.

These are preview downloads. The Mac app is locally/ad-hoc signed, not Developer
ID signed or notarized, and the Windows executables are unsigned. Your system
may warn about an unknown publisher. Download only from this repository, verify
`SHA256SUMS.txt`, and do not disable security protections to run the app.

On Mac, grant **Input Monitoring** to the final Applications copy and reopen it
if macOS requests it. If Gatekeeper blocks it and you trust the download, use
the system's **Privacy & Security → Open Anyway** option when available.

The Windows preview includes all ten presets, intensity, volume, mute, release
sounds, a native studio, a tray icon when available and a three-second counter. Its floating
mouse-following HUD and automatic login setup are not yet ported.

CI builds and checks the app on Intel macOS, Apple-silicon macOS, Windows x64 and
Windows ARM64, including the production sound bank. Windows smoke tests check
native controls, hook installation, close/reopen and shutdown without an audio
device. Windows x64 passes a strict tray check. **Known ARM64 limitation:** tray
registration failed on its CI runner even with Explorer present. The experimental
ARM64 build instead passes the minimize-to-taskbar/reopen fallback check; its
tray integration is not verified. Close minimizes if no tray is available.
This is not a guarantee of global keyboard access or audio playback on every
physical device: please test on your target computer and report issues.

No typed text is saved or transmitted. Only aggregate counts, transient key-down
state and timestamps are retained in memory. Protected/secure desktops and
security policies can prevent observation of keyboard events.
