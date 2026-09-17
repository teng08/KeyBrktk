package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Keybed", "settings.json")
	if readSettings(path) != defaultSettings() {
		t.Fatal("missing settings should use defaults")
	}
	want := settings{Preset: len(presetNames) - 1, Intensity: 2, Volume: 0, Muted: true, Releases: false}
	if err := saveSettings(path, want); err != nil {
		t.Fatal(err)
	}
	if readSettings(path) != want {
		t.Fatal("settings did not survive restart")
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatal("temporary files were left behind")
	}
	if err := os.WriteFile(path, []byte("broken JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if readSettings(path) != defaultSettings() {
		t.Fatal("corrupt settings should use defaults")
	}
	if err := os.WriteFile(path, []byte(`{"preset":999,"intensity":-1,"volume":999}`), 0600); err != nil {
		t.Fatal(err)
	}
	got := readSettings(path)
	if got.Preset != 0 || got.Intensity != 1 || got.Volume != 100 {
		t.Fatal("invalid settings were not bounded")
	}
}
