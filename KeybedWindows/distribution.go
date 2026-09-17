package main

import (
	"bytes"
	"path/filepath"
)

// Release builds populate these through embedded_bundled.go. Ordinary source
// builds remain usable with a sidecar sound bank and need no generated assets.
var bundledSoundBank []byte
var bundledCredits string

func loadAppBank(overridePath, executable string) (*soundBank, error) {
	return selectAppBank(overridePath, executable, bundledSoundBank)
}

func selectAppBank(overridePath, executable string, embedded []byte) (*soundBank, error) {
	if overridePath != "" {
		return loadBank(overridePath)
	}
	if len(embedded) != 0 {
		return readBank(bytes.NewReader(embedded))
	}
	return loadBank(filepath.Join(filepath.Dir(executable), "Keybed.soundbank"))
}

func appCredits() string {
	if bundledCredits != "" {
		return bundledCredits
	}
	return "Source build: sound credits and full license notices are in Sounds/CREDITS.md, Sounds/Alpaca/LICENSE and Sounds/Mechanical/LICENSE. Release executables embed these notices."
}
