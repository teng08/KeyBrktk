package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDistributionBankSelection(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "Keybed.exe")
	data := bankBytes(testBank())
	bank, err := selectAppBank("", executable, data)
	if err != nil || bank == nil {
		t.Fatalf("standalone embedded load: %v", err)
	}
	// An explicit override must win, even when it is missing or invalid.
	if _, err := selectAppBank(filepath.Join(dir, "missing"), executable, data); err == nil {
		t.Fatal("missing explicit override silently fell back to embedded data")
	}
	if err := os.WriteFile(filepath.Join(dir, "Keybed.soundbank"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := selectAppBank("", executable, nil); err != nil {
		t.Fatalf("source-build sidecar fallback: %v", err)
	}
	if _, err := selectAppBank("", executable, []byte("broken")); err == nil {
		t.Fatal("corrupt embedded data silently fell back to the sidecar")
	}
	if _, err := selectAppBank(filepath.Join(dir, "Keybed.soundbank"), executable, []byte("broken")); err != nil {
		t.Fatalf("explicit override did not win: %v", err)
	}
}
