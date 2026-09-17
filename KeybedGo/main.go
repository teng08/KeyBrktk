// Keybed's optional Go launcher. All keyboard/audio work runs in the native app.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func projectRoot() (string, error) {
	executable, _ := os.Executable()
	workingDirectory, _ := os.Getwd()
	_, source, _, _ := runtime.Caller(0)
	candidates := []string{filepath.Dir(executable), filepath.Dir(filepath.Dir(executable)),
		workingDirectory, filepath.Dir(workingDirectory), filepath.Dir(filepath.Dir(source))}
	for _, root := range candidates {
		if _, err := os.Stat(filepath.Join(root, "KeybedMac", "build-app.sh")); err == nil {
			return root, nil
		}
		if _, err := os.Stat(filepath.Join(root, "Keybed.app", "Contents", "MacOS", "Keybed")); err == nil {
			return root, nil
		}
	}
	return "", fmt.Errorf("cannot find Keybed.app; run from the keyboard sound project folder")
}

func run() error {
	rebuild := flag.Bool("build", false, "rebuild the native app before opening")
	selfTest := flag.Bool("self-test", false, "verify bundled sounds without opening a window")
	flag.Parse()
	app := "/Applications/Keybed.app"
	binary := filepath.Join(app, "Contents", "MacOS", "Keybed")
	if _, err := os.Stat(binary); *rebuild || err != nil {
		root, err := projectRoot()
		if err != nil {
			return err
		}
		app = filepath.Join(root, "Keybed.app")
		binary = filepath.Join(app, "Contents", "MacOS", "Keybed")
		if _, err := os.Stat(binary); *rebuild || os.IsNotExist(err) {
			build := exec.Command("/bin/zsh", filepath.Join(root, "KeybedMac", "build-app.sh"))
			build.Stdout, build.Stderr = os.Stdout, os.Stderr
			if err := build.Run(); err != nil {
				return fmt.Errorf("build Keybed: %w", err)
			}
		}
	}
	if *selfTest {
		check := exec.Command(binary, "--self-test")
		check.Stdout, check.Stderr = os.Stdout, os.Stderr
		return check.Run()
	}
	return exec.Command("/usr/bin/open", app).Run()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Keybed:", err)
		os.Exit(1)
	}
}
