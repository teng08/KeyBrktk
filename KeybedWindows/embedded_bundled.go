//go:build bundled

package main

import "embed"

// Assets are staged by build.ps1, never downloaded at runtime.
//
//go:embed assets/Keybed.soundbank
var releaseSoundBank []byte

//go:embed assets/CREDITS.md assets/Alpaca-LICENSE assets/Mechanical-LICENSE assets/SOURCES.json
var releaseNotices embed.FS

func init() {
	bundledSoundBank = releaseSoundBank
	for _, notice := range []string{"CREDITS.md", "Alpaca-LICENSE", "Mechanical-LICENSE", "SOURCES.json"} {
		data, err := releaseNotices.ReadFile("assets/" + notice)
		if err != nil {
			panic(err) // A corrupt build must not silently omit copyright notices.
		}
		bundledCredits += "\n\n--- " + notice + " ---\n\n" + string(data)
	}
}
