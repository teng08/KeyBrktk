//go:build bundled

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedDistribution(t *testing.T) {
	// There is intentionally no Keybed.soundbank beside this fictitious EXE.
	bank, err := loadAppBank("", filepath.Join(t.TempDir(), "Keybed.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBank(bank); err != nil {
		t.Fatal(err)
	}
	for name, source := range map[string]string{
		"CREDITS.md":         "../Sounds/CREDITS.md",
		"Alpaca-LICENSE":     "../Sounds/Alpaca/LICENSE",
		"Mechanical-LICENSE": "../Sounds/Mechanical/LICENSE",
		"SOURCES.json":       "../Sounds/Mechanical/SOURCES.json",
	} {
		want, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		got, err := releaseNotices.ReadFile("assets/" + name)
		if err != nil || !bytes.Equal(got, want) || !strings.Contains(appCredits(), string(want)) {
			t.Fatalf("embedded notice differs from source: %s (%v)", name, err)
		}
	}
	if source := os.Getenv("KEYBED_SOUND_BANK"); source != "" {
		want, err := os.ReadFile(source)
		if err != nil || !bytes.Equal(bundledSoundBank, want) {
			t.Fatalf("embedded sound bank differs from production export: %v", err)
		}
	}
}
