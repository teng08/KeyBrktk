## Downloads

Windows now has the **same dark Keybed Sound Studio design as Mac**: matching
colored sound cards, descriptions, selected badges, the featured Skibiddy card,
scrollable library, intensity buttons, volume, release sounds, mute/preview and
reset count. This update also ports the click-through mouse-following floating
counter and automatic-login checkbox to Windows. The Windows studio handles
high-DPI displays and short-screen scrolling. Native OS controls/title bars and
keyboard permission integration retain platform differences.

Both desktop apps include 13 presets and 468 preloaded sounds, including recorded
Holy Pandas, NovelKeys Creams and Turquoise Tealios, dedicated Space/Enter/Backspace
and releases. Previous saved sound selections retain their meaning.

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

Windows login startup is off by default; enable it after placing the extracted
folder in its permanent location. The floating counter is optional and its
visibility persists. Close keeps the sounds and overlay running; use Quit to stop.

CI builds and checks the app on Intel macOS, Apple-silicon macOS, Windows x64 and
Windows ARM64, including all 468 production sounds, recording hashes/licenses,
matching preset catalogs and settings compatibility. Mac smoke tests check all
preset selections, scrolling, HUD/counting and close/reopen with audio output
disabled. Windows smoke tests click every sound card and intensity control,
verify small-screen scrolling/DPI changes, floating counting/idle collapse/reset
and focus, and check hook installation, close/reopen and shutdown without an audio
device. Windows x64 passes a strict tray check. **Known ARM64 limitation:** tray
registration failed on its CI runner even with Explorer present. The experimental
ARM64 build instead passes the minimize-to-taskbar/reopen fallback check; its
tray integration is not verified. Close minimizes if no tray is available.
This is not a guarantee of global keyboard access or audio playback on every
physical device: please test on your target computer and report issues.
Native screenshots from each target are available as CI workflow artifacts.

No typed text is saved or transmitted. Only aggregate counts, transient key-down
state and timestamps are retained in memory. Protected/secure desktops and
security policies can prevent observation of keyboard events.
