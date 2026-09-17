package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestDesktopPresetCatalogsMatch(t *testing.T) {
	data, err := os.ReadFile("../KeybedMac/Sources/KeybedMac/SoundPreset.swift")
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`\.init\(id: "([^"]+)", name: "([^"]+)"`).FindAllSubmatch(data, -1)
	if len(matches) != len(presetNames) {
		t.Fatalf("Mac has %d presets; Windows has %d", len(matches), len(presetNames))
	}
	for index, match := range matches {
		if string(match[2]) != presetNames[index] {
			t.Fatalf("preset %d differs between Mac and Windows", index)
		}
	}
	// Existing Windows settings store numeric indices, so keep this prefix stable.
	original := [...]string{"Alpaca Linear", "Tactile Brown", "Clicky Blue", "Deep Thock", "Creamy Marble", "Typewriter", "Bubble Pop", "Pixel Tap", "Birdy Chirp", "Skibiddy Toilet"}
	for index, name := range original {
		if presetNames[index] != name {
			t.Fatalf("saved preset %d would change meaning", index)
		}
	}
}

func TestRecordedSwitchAssets(t *testing.T) {
	var manifest struct {
		SourceRepository, SourceCommit string
		Presets                        []struct {
			ID, Directory, Name string
			Files               []struct {
				File, SourcePath, GitBlobSHA string
				Bytes                        int
			}
		}
	}
	data, err := os.ReadFile("../Sounds/Mechanical/SOURCES.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceRepository != "https://github.com/tplai/kbsim" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(manifest.SourceCommit) {
		t.Fatal("recording source must reference a pinned upstream commit")
	}
	directories := [...]string{"HolyPanda", "NovelKeysCreams", "TurquoiseTealios"}
	ids := [...]string{"holypanda", "cream", "turquoise"}
	files := [...]string{"press_key1.mp3", "press_key2.mp3", "press_key3.mp3", "press_key4.mp3", "press_key5.mp3", "press_space.mp3", "press_enter.mp3", "press_back.mp3", "release_key.mp3", "release_space.mp3", "release_enter.mp3", "release_back.mp3"}
	if len(manifest.Presets) != len(directories) {
		t.Fatal("missing requested switch recordings")
	}
	for index, preset := range manifest.Presets {
		if preset.ID != ids[index] || preset.Directory != directories[index] || preset.Name != presetNames[10+index] || len(preset.Files) != len(files) {
			t.Fatalf("incomplete or mismatched recording pack: %s", preset.Name)
		}
		for key, file := range preset.Files {
			if file.File != files[key] {
				t.Fatal("recording key mapping differs from the production bank")
			}
			path := filepath.Join("..", "Sounds", preset.Directory, file.File)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) != file.Bytes || len(data) == 0 {
				t.Fatalf("recording is missing or truncated: %s", path)
			}
			hash := sha1.New()
			fmt.Fprintf(hash, "blob %d\x00", len(data))
			hash.Write(data)
			if hex.EncodeToString(hash.Sum(nil)) != file.GitBlobSHA {
				t.Fatalf("recording differs from the pinned upstream Git blob: %s", path)
			}
		}
	}
	license, err := os.ReadFile("../Sounds/Mechanical/LICENSE")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha1.New()
	fmt.Fprintf(hash, "blob %d\x00", len(license))
	hash.Write(license)
	if hex.EncodeToString(hash.Sum(nil)) != "69da5f1199abad0b9dc9b8063ac43a417ff877d5" {
		t.Fatal("the upstream recording license must be included unchanged")
	}
}
