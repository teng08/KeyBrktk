# Keyboard sound credits

Keybed's native app uses the bundled **Alpaca Linear** press and release recordings from
[Thomas Lai's kbsim](https://github.com/tplai/kbsim), distributed under the MIT
license. The full copyright notice and license are included in [Alpaca/LICENSE](Alpaca/LICENSE).

**Holy Pandas, NovelKeys Creams and Turquoise Tealios** also use Thomas Lai's
recorded switch samples from kbsim under the MIT license. Each pack has five
normal-key press variants plus separate Space, Enter, Backspace and release
recordings. The original files are bundled without modification; desktop apps
trim and process them in memory before playback. These are recordings of
particular keyboard setups, not a promise that every physical switch sounds
identical.

The pinned source commit, upstream paths, file sizes and Git blob hashes for all
36 new recordings are in [Mechanical/SOURCES.json](Mechanical/SOURCES.json).
The upstream copyright notice and full license are in
[Mechanical/LICENSE](Mechanical/LICENSE), included in both desktop downloads.

**Tactile Brown** uses the four `stav-tactile` recordings. **Clicky Blue** uses
the BlueSwitch recordings. Their key variants, pitch and release envelopes are
prepared in memory when the app starts. Clicky / typing recordings remain
available for the browser demo.

These recordings are from Freesound packs marked **Creative Commons 0 (CC0)**.

- Tactile: StavSounds, [Mechanical Keyboards pack 42151](https://freesound.org/people/StavSounds/packs/42151/)
- BlueSwitch: UberBosser, [mechanicalKeys pack 23846](https://freesound.org/people/UberBosser/packs/23846/)
- Clicky / typing: Capt.Jack, [Mechanical Keyboard pack 43809](https://freesound.org/people/Capt.Jack/packs/43809/)

The app uses Freesound preview files bundled locally for playback.

**Deep Thock, Creamy Marble, Typewriter, Bubble Pop, Pixel Tap and Birdy Chirp**
are generated locally by Keybed's sound synthesizer. They contain no additional
downloaded recordings and need no network connection.

**Skibiddy Toilet** uses locally generated robot vocal samples rendered with
macOS's Fred voice, plus synthetic release pops. The generation script and
voice phrases are in [Skibiddy/README.md](Skibiddy/README.md).
