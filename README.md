# Keybed

For Mac integration, install Keybed into Applications:

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

The **Sound Studio** has ten presets: **Alpaca Linear, Tactile Brown, Clicky Blue,
Deep Thock, Creamy Marble, Typewriter, Bubble Pop, Pixel Tap, Birdy Chirp and Skibiddy Toilet**.
Click a sound card to select and preview it, or choose a sound from the menu bar.
New presets have a **NEW** badge. Try the text field to hear actual typing.
The three mechanical presets use bundled recordings; six presets are
synthesized at startup. **Skibiddy Toilet** uses generated robot vocals saying
“skibiddy”, “toilet”, “dop dop” and “yes yes”. Enter plays “skibiddy toilet”.
All sounds work offline and are preloaded; speech is never generated on a key press.

The animated floating counter follows your **mouse pointer**, displays the
current sound and counts key-down events, including held-key repeats. It is
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
memory. All 360 preset/intensity/key combinations are preloaded. MP3 decoding,
synthesis, silence trimming, normalization and tone processing happen at startup.
A dedicated keyboard event thread feeds the audio callback, which
does not allocate memory or wait for a mutex. Playback starts in the next audio
block; actual audible latency depends on the output device. Bluetooth audio adds
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
or external Go module is required. The script builds for the current Mac's CPU.

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
