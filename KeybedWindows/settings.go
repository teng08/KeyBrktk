package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type settings struct {
	Preset    int  `json:"preset"`
	Intensity int  `json:"intensity"`
	Volume    int  `json:"volume"`
	Muted     bool `json:"muted"`
	Releases  bool `json:"releases"`
	Overlay   bool `json:"overlay"`
}

func defaultSettings() settings {
	return settings{Intensity: 1, Volume: 72, Releases: true, Overlay: true}
}
func (s *settings) sanitize() {
	if s.Preset < 0 || s.Preset >= len(presetNames) {
		s.Preset = 0
	}
	if s.Intensity < 0 || s.Intensity >= len(intensityNames) {
		s.Intensity = 1
	}
	if s.Volume < 0 {
		s.Volume = 0
	}
	if s.Volume > 100 {
		s.Volume = 100
	}
}

func settingsPath() string {
	directory, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(directory, "Keybed", "settings.json")
}

func readSettings(path string) settings {
	s := defaultSettings()
	data, err := os.ReadFile(path)
	if err == nil {
		candidate := s
		if json.Unmarshal(data, &candidate) == nil {
			s = candidate
		}
	}
	s.sanitize()
	return s
}

func saveSettings(path string, s settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".settings-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
