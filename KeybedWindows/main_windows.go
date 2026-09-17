//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	windowClassName = "Keybed.SoundStudio"
	trayMessage     = 0x8001
	idVolume        = 102
	idMute          = 103
	idReleases      = 104
	idPreview       = 105
	idQuit          = 106
	idShow          = 107
)

type studio struct {
	window, volume uintptr
	studioUI
	mute, releases, volumeLabel      uintptr
	status, count, taskbarMessage    uintptr
	icon                             iconData
	settings                         settings
	mixer                            *mixer
	counter                          typingCounter
	output                           *outputState
	smoke                            bool
	allowTrayFallback, trayAvailable bool
	lastTrayAttempt                  time.Time
	smokeError                       error
}

func (s *studio) apply() {
	s.mixer.configure(s.settings)
	s.refreshControls()
	if !s.smoke {
		if err := saveSettings(settingsPath(), s.settings); err != nil {
			setText(s.status, "Could not save settings: "+err.Error())
		}
	}
}

func (s *studio) toggleMute() {
	s.settings.Muted = !s.settings.Muted
	s.apply()
}

func (s *studio) show() { showWindow.Call(s.window, 9); setForegroundWindow.Call(s.window) }

func (s *studio) addTray() bool {
	s.lastTrayAttempt = time.Now()
	result, _, _ := notifyIcon.Call(0, uintptr(unsafe.Pointer(&s.icon)))
	s.trayAvailable = result != 0
	return s.trayAvailable
}

func (s *studio) menu() {
	menu, _, _ := createPopupMenu.Call()
	if menu == 0 {
		return
	}
	defer destroyMenu.Call(menu)
	appendMenu.Call(menu, 0, idShow, uintptr(unsafe.Pointer(wide("Show Keybed"))))
	label := "Mute"
	if s.settings.Muted {
		label = "Unmute"
	}
	appendMenu.Call(menu, 0, idMute, uintptr(unsafe.Pointer(wide(label))))
	appendMenu.Call(menu, 0, idPreview, uintptr(unsafe.Pointer(wide("Test sound"))))
	appendMenu.Call(menu, 0x800, 0, 0)
	appendMenu.Call(menu, 0, idQuit, uintptr(unsafe.Pointer(wide("Quit Keybed"))))
	var point [2]int32
	getCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	setForegroundWindow.Call(s.window)
	trackPopupMenu.Call(menu, 0x02, uintptr(point[0]), uintptr(point[1]), 0, s.window, 0)
}

func (s *studio) procedure(window, message, wparam, lparam uintptr) uintptr {
	if result, handled := s.uiMessage(window, message, wparam, lparam); handled {
		return result
	}
	if message != 0 && message == s.taskbarMessage {
		// Restore the tray icon when Explorer/the taskbar is restarted.
		s.addTray()
		return 0
	}
	switch message {
	case 0x10: // Close hides the studio; the explicit Quit command stops listening.
		if s.trayAvailable {
			showWindow.Call(window, 0)
		} else {
			showWindow.Call(window, 6)
		}
		return 0
	case 0x02:
		notifyIcon.Call(2, uintptr(unsafe.Pointer(&s.icon)))
		s.disposeUI()
		postQuitMessage.Call(0)
		return 0
	case 0x111:
		id, notification := wparam&0xffff, (wparam>>16)&0xffff
		if notification == 6 {
			for _, control := range s.controls {
				if control.window == lparam {
					top, bottom := s.px(control.rect.y), s.px(control.rect.y+control.rect.h)
					page := int(clientRect(s.window).bottom)
					if top < s.rootScroll {
						s.rootScroll = top
					}
					if bottom > s.rootScroll+page {
						s.rootScroll = bottom - page
					}
					s.layoutStudio()
					break
				}
			}
			return 0
		}
		if notification != 0 {
			return 0
		}
		switch id {
		case idMute:
			s.toggleMute()
		case idReleases:
			value, _, _ := sendMessage.Call(s.releases, 0xf0, 0, 0)
			s.settings.Releases = value == 1
			s.apply()
		case idOverlay:
			checked, _, _ := sendMessage.Call(s.overlay, 0xf0, 0, 0)
			s.settings.Overlay = checked == 1
			s.apply()
		case idReset:
			s.counter.reset()
			setText(s.count, "0")
			s.previewTime = time.Time{}
			s.tickHUD(time.Now())
		case idStartup:
			checked, _, _ := sendMessage.Call(s.startup, 0xf0, 0, 0)
			if !s.smoke {
				if err := setLoginEnabled(checked == 1); err != nil {
					setText(s.status, err.Error())
				} else {
					s.startupEnabled = loginEnabled()
				}
			}
			s.refreshControls()
		case idPreview:
			s.mixer.enqueue(keyEvent{})
			s.previewTime = time.Now()
			s.tickHUD(time.Now())
		case idQuit:
			destroyWindow.Call(window)
		case idShow:
			s.show()
		default:
			if notification == 0 && id >= idCardBase && id < idCardBase+uintptr(len(presetNames)) {
				s.settings.Preset = int(id - idCardBase)
				s.apply()
				s.mixer.enqueue(keyEvent{})
				s.previewTime = time.Now()
				s.tickHUD(time.Now())
			} else if notification == 0 && id >= idModeBase && id < idModeBase+uintptr(len(intensityNames)) {
				s.settings.Intensity = int(id - idModeBase)
				s.apply()
				s.mixer.enqueue(keyEvent{})
			}
		}
		return 0
	case 0x114:
		if lparam == s.volume {
			value, _, _ := sendMessage.Call(s.volume, 0x400, 0, 0)
			s.settings.Volume = int(value)
			setText(s.volumeLabel, fmt.Sprintf("Volume: %d%%", value))
			s.apply()
		}
		return 0
	case 0x113:
		if wparam == 3 {
			s.tickHUD(time.Now())
			return 0
		}
		if wparam == 2 {
			s.smokeError = s.checkStudio()
			destroyWindow.Call(window)
			return 0
		}
		setText(s.count, fmt.Sprintf("%d", s.counter.snapshot(time.Now())))
		status := "Audio disabled for smoke test"
		if s.output != nil {
			status = s.output.status.Load().(string)
		}
		if s.settings.Muted {
			status = "Muted · keyboard listener still active"
		}
		if !s.trayAvailable {
			if time.Since(s.lastTrayAttempt) >= 2*time.Second {
				s.addTray()
			}
			if !s.trayAvailable {
				status = "No tray; Close minimizes · " + status
			}
		}
		setText(s.status, status)
		return 0
	case trayMessage:
		switch lparam {
		case 0x203:
			s.show()
		case 0x205, 0x7b:
			s.menu()
		}
		return 0
	}
	result, _, _ := defWindowProc.Call(window, message, wparam, lparam)
	return result
}

func (s *studio) create() error {
	s.taskbarMessage, _, _ = registerMessage.Call(uintptr(unsafe.Pointer(wide("TaskbarCreated"))))
	common := [2]uint32{8, 4}
	result, _, err := syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&common)))
	if result == 0 {
		return winError("initialize controls", err)
	}
	return s.createStudioUI()
}

func run() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	selfTest := flag.Bool("self-test", false, "test all sounds without an audio device or keyboard hook")
	smoke := flag.Bool("smoke-test", false, "test the window, tray and keyboard listener, then exit")
	background := flag.Bool("background", false, "start in the tray without showing the studio")
	screenshot := flag.String("screenshot", "", "save the studio UI during a smoke test only")
	noAudio := flag.Bool("no-audio", false, "disable audio for the smoke test only")
	allowTrayFallback := flag.Bool("allow-tray-fallback", false, "test the documented experimental ARM64 no-tray fallback")
	bankPath := flag.String("bank", filepath.Join(filepath.Dir(executable), "Keybed.soundbank"), "preloaded sound bank path")
	flag.Parse()
	if *screenshot != "" && !*smoke {
		return fmt.Errorf("--screenshot requires --smoke-test")
	}
	if *noAudio && !*smoke {
		return fmt.Errorf("--no-audio is only supported with --smoke-test")
	}
	if *allowTrayFallback && (!*smoke || runtime.GOARCH != "arm64") {
		return fmt.Errorf("--allow-tray-fallback is only supported with --smoke-test on experimental Windows ARM64")
	}
	bank, err := loadBank(*bankPath)
	if err != nil {
		return err
	}
	if *selfTest {
		return verifyBank(bank)
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if setDPIAwareness.Find() == nil {
		setDPIAwareness.Call(^uintptr(3))
	} else {
		user32.NewProc("SetProcessDPIAware").Call()
	}
	comResult, _, _ := initializeCOM.Call(0, 2) // STA for Shell APIs on the UI thread.
	if int32(comResult) < 0 {
		return fmt.Errorf("initialize desktop COM: %#x", uint32(comResult))
	}
	defer uninitializeCOM.Call()
	mutex, _, mutexError := createMutex.Call(0, 0, uintptr(unsafe.Pointer(wide("Local\\Keybed.KeyboardSound"))))
	if mutex == 0 {
		return winError("create single-instance guard", mutexError)
	}
	defer closeHandle.Call(mutex)
	if mutexError == syscall.Errno(183) {
		window, _, _ := findWindow.Call(uintptr(unsafe.Pointer(wide(windowClassName))), 0)
		if window != 0 {
			showWindow.Call(window, 9)
			setForegroundWindow.Call(window)
		}
		return nil
	}
	s := &studio{settings: readSettings(settingsPath()), mixer: newMixer(bank), smoke: *smoke, allowTrayFallback: *allowTrayFallback}
	s.screenshotPath = *screenshot
	s.mixer.configure(s.settings)
	if err := s.create(); err != nil {
		if s.window != 0 {
			destroyWindow.Call(s.window)
		}
		return err
	}
	listener, err := startKeyboard(s.mixer, &s.counter)
	if err != nil {
		destroyWindow.Call(s.window)
		return err
	}
	defer listener.stop()
	stopAudio := make(chan struct{})
	if !*noAudio {
		s.output = startOutput(s.mixer, stopAudio)
		defer func() { close(stopAudio); <-s.output.done }()
	}
	if !*background || *smoke {
		showWindow.Call(s.window, 5)
	} else if !s.trayAvailable {
		showWindow.Call(s.window, 6)
	}
	if *smoke {
		// Give Explorer registration retries time to run while pumping messages.
		setTimer.Call(s.window, 2, 6000, 0)
	}
	var message windowMessage
	for {
		result, _, err := getMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			destroyWindow.Call(s.window)
			return winError("studio message loop", err)
		}
		if result == 0 {
			break
		}
		if handled, _, _ := isDialogMessage.Call(s.window, uintptr(unsafe.Pointer(&message))); handled != 0 {
			continue
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	if *smoke {
		if s.smokeError != nil {
			return s.smokeError
		}
		if s.trayAvailable {
			fmt.Println("PASS: matching dark sound-card studio, scrolling, floating counter, controls, tray icon, global keyboard hook and clean shutdown.")
		} else {
			fmt.Println("PASS: matching dark sound-card studio, scrolling, floating counter, controls, minimize-to-taskbar/reopen fallback, global keyboard hook and clean shutdown. EXPERIMENTAL ARM64 LIMITATION: tray registration did not pass.")
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Keybed:", err)
		// Command-line validation must fail cleanly rather than blocking CI on a dialog.
		validation := false
		for _, name := range []string{"self-test", "smoke-test"} {
			if option := flag.Lookup(name); option != nil && option.Value.String() == "true" {
				validation = true
			}
		}
		if !validation {
			showError(err.Error())
		}
		os.Exit(1)
	}
}
