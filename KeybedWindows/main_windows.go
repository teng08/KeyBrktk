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
	idPreset        = 100
	idIntensity     = 101
	idVolume        = 102
	idMute          = 103
	idReleases      = 104
	idPreview       = 105
	idQuit          = 106
	idShow          = 107
)

type studio struct {
	window, preset, intensity, volume uintptr
	mute, releases, volumeLabel       uintptr
	status, count, taskbarMessage     uintptr
	icon                              iconData
	settings                          settings
	mixer                             *mixer
	counter                           typingCounter
	output                            *outputState
	smoke                             bool
	allowMissingTray, trayAvailable   bool
	lastTrayAttempt                   time.Time
	smokeError                        error
}

func (s *studio) apply() {
	s.mixer.configure(s.settings)
	if !s.smoke {
		if err := saveSettings(settingsPath(), s.settings); err != nil {
			setText(s.status, "Could not save settings: "+err.Error())
		}
	}
}

func (s *studio) toggleMute() {
	s.settings.Muted = !s.settings.Muted
	value := uintptr(0)
	if s.settings.Muted {
		value = 1
	}
	sendMessage.Call(s.mute, 0xf1, value, 0)
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
		postQuitMessage.Call(0)
		return 0
	case 0x111:
		id, notification := wparam&0xffff, (wparam>>16)&0xffff
		switch id {
		case idPreset:
			if notification == 1 {
				value, _, _ := sendMessage.Call(s.preset, 0x147, 0, 0)
				s.settings.Preset = int(value)
				s.apply()
				s.mixer.enqueue(keyEvent{})
			}
		case idIntensity:
			if notification == 1 {
				value, _, _ := sendMessage.Call(s.intensity, 0x147, 0, 0)
				s.settings.Intensity = int(value)
				s.apply()
				s.mixer.enqueue(keyEvent{})
			}
		case idMute:
			s.toggleMute()
		case idReleases:
			value, _, _ := sendMessage.Call(s.releases, 0xf0, 0, 0)
			s.settings.Releases = value == 1
			s.apply()
		case idPreview:
			s.mixer.enqueue(keyEvent{})
		case idQuit:
			destroyWindow.Call(window)
		case idShow:
			s.show()
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
		if wparam == 2 {
			s.smokeError = s.checkStudio()
			destroyWindow.Call(window)
			return 0
		}
		setText(s.count, fmt.Sprintf("%d keys in this 3-second window", s.counter.snapshot(time.Now())))
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

func (s *studio) control(class, text string, style uintptr, x, y, width, height int, id uintptr) (uintptr, error) {
	module, _, _ := getModuleHandle.Call(0)
	window, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), style|0x50000000,
		uintptr(x), uintptr(y), uintptr(width), uintptr(height), s.window, id, module, 0)
	if window == 0 {
		return 0, winError("create "+class, err)
	}
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("GetStockObject").Call(17)
	sendMessage.Call(window, 0x30, font, 1)
	return window, nil
}

// Exercise native controls and close/reopen behavior, not just their creation.
func (s *studio) checkStudio() error {
	count, _, _ := sendMessage.Call(s.preset, 0x146, 0, 0)
	if int(count) != len(presetNames) {
		return fmt.Errorf("preset dropdown is incomplete")
	}
	count, _, _ = sendMessage.Call(s.intensity, 0x146, 0, 0)
	if int(count) != len(intensityNames) {
		return fmt.Errorf("intensity dropdown is incomplete")
	}
	sendMessage.Call(s.preset, 0x14e, 9, 0)
	sendMessage.Call(s.window, 0x111, (1<<16)|idPreset, s.preset)
	if s.settings.Preset != 9 || s.mixer.preset.Load() != 9 {
		return fmt.Errorf("preset control did not configure audio")
	}
	sendMessage.Call(s.intensity, 0x14e, 2, 0)
	sendMessage.Call(s.window, 0x111, (1<<16)|idIntensity, s.intensity)
	if s.settings.Intensity != 2 || s.mixer.intensity.Load() != 2 {
		return fmt.Errorf("intensity control did not configure audio")
	}
	sendMessage.Call(s.volume, 0x405, 1, 61)
	sendMessage.Call(s.window, 0x114, 0, s.volume)
	if s.settings.Volume != 61 {
		return fmt.Errorf("volume slider did not configure audio")
	}
	sendMessage.Call(s.releases, 0xf1, 0, 0)
	sendMessage.Call(s.window, 0x111, idReleases, s.releases)
	if s.settings.Releases || s.mixer.releases.Load() {
		return fmt.Errorf("release control did not configure audio")
	}
	wasMuted := s.settings.Muted
	sendMessage.Call(s.window, 0x111, idMute, s.mute)
	if s.settings.Muted == wasMuted || s.mixer.muted.Load() != s.settings.Muted {
		return fmt.Errorf("mute control did not configure audio")
	}
	sendMessage.Call(s.window, 0x10, 0, 0)
	if s.trayAvailable {
		visible, _, _ := isWindowVisible.Call(s.window)
		if visible != 0 {
			return fmt.Errorf("close did not hide the studio")
		}
	} else {
		minimized, _, _ := isIconic.Call(s.window)
		if minimized == 0 {
			return fmt.Errorf("close did not minimize without a tray")
		}
	}
	s.show()
	visible, _, _ := isWindowVisible.Call(s.window)
	minimized, _, _ := isIconic.Call(s.window)
	if visible == 0 || minimized != 0 {
		return fmt.Errorf("studio did not reopen")
	}
	return nil
}

func (s *studio) create() error {
	s.taskbarMessage, _, _ = registerMessage.Call(uintptr(unsafe.Pointer(wide("TaskbarCreated"))))
	common := [2]uint32{8, 4}
	result, _, err := syscall.NewLazyDLL("comctl32.dll").NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&common)))
	if result == 0 {
		return winError("initialize controls", err)
	}
	module, _, _ := getModuleHandle.Call(0)
	icon, _, _ := loadIcon.Call(0, 32512)
	cursor, _, _ := loadCursor.Call(0, 32512)
	class := windowClass{procedure: syscall.NewCallback(s.procedure), instance: module, icon: icon, cursor: cursor, background: 16, className: wide(windowClassName)}
	result, _, err = registerClass.Call(uintptr(unsafe.Pointer(&class)))
	if result == 0 {
		return winError("register studio window", err)
	}
	s.window, _, err = createWindow.Call(0, uintptr(unsafe.Pointer(class.className)), uintptr(unsafe.Pointer(wide("Keybed · Sound Studio"))), 0x00ca0000,
		160, 120, 540, 485, 0, 0, module, 0)
	if s.window == 0 {
		return winError("create studio window", err)
	}
	// Return errors for missing controls rather than quietly shipping a broken UI.
	add := func(class, text string, style uintptr, x, y, w, h int, id uintptr, target *uintptr) error {
		handle, err := s.control(class, text, style, x, y, w, h, id)
		if target != nil {
			*target = handle
		}
		return err
	}
	var controls = []struct {
		class, text string
		style       uintptr
		x, y, w, h  int
		id          uintptr
		target      *uintptr
	}{
		{"STATIC", "Keyboard sounds, everywhere you type", 0, 24, 20, 480, 24, 0, nil},
		{"STATIC", "Sound preset", 0, 24, 58, 150, 22, 0, nil},
		{"COMBOBOX", "", 0x00210003, 24, 82, 475, 300, idPreset, &s.preset},
		{"STATIC", "Intensity", 0, 24, 124, 150, 22, 0, nil},
		{"COMBOBOX", "", 0x00210003, 24, 148, 475, 180, idIntensity, &s.intensity},
		{"STATIC", fmt.Sprintf("Volume: %d%%", s.settings.Volume), 0, 24, 193, 180, 22, 0, &s.volumeLabel},
		{"msctls_trackbar32", "", 0x10001, 20, 218, 480, 36, idVolume, &s.volume},
		{"BUTTON", "Mute", 0x10003, 24, 268, 190, 24, idMute, &s.mute},
		{"BUTTON", "Key release sounds", 0x10003, 245, 268, 250, 24, idReleases, &s.releases},
		{"BUTTON", "Test sound", 0x10000, 24, 309, 185, 35, idPreview, nil},
		{"BUTTON", "Quit Keybed", 0x10000, 315, 309, 185, 35, idQuit, nil},
		{"STATIC", "Starting…", 0, 24, 363, 480, 35, 0, &s.status},
		{"STATIC", "", 0, 24, 402, 480, 22, 0, &s.count},
	}
	for _, c := range controls {
		if err := add(c.class, c.text, c.style, c.x, c.y, c.w, c.h, c.id, c.target); err != nil {
			return err
		}
	}
	for _, name := range presetNames {
		sendMessage.Call(s.preset, 0x143, 0, uintptr(unsafe.Pointer(wide(name))))
	}
	for _, name := range intensityNames {
		sendMessage.Call(s.intensity, 0x143, 0, uintptr(unsafe.Pointer(wide(name))))
	}
	sendMessage.Call(s.preset, 0x14e, uintptr(s.settings.Preset), 0)
	sendMessage.Call(s.intensity, 0x14e, uintptr(s.settings.Intensity), 0)
	sendMessage.Call(s.volume, 0x406, 1, 100<<16)
	sendMessage.Call(s.volume, 0x405, 1, uintptr(s.settings.Volume))
	if s.settings.Muted {
		sendMessage.Call(s.mute, 0xf1, 1, 0)
	}
	if s.settings.Releases {
		sendMessage.Call(s.releases, 0xf1, 1, 0)
	}
	s.icon = iconData{size: uint32(unsafe.Sizeof(iconData{})), window: s.window, id: 1, flags: 7, callback: trayMessage, icon: icon}
	copy(s.icon.tip[:], syscall.StringToUTF16("Keybed · keyboard sounds"))
	if !s.addTray() {
		taskbar, _, _ := findWindow.Call(uintptr(unsafe.Pointer(wide("Shell_TrayWnd"))), 0)
		if s.smoke && (!s.allowMissingTray || taskbar != 0) {
			return fmt.Errorf("create notification-area icon failed (Explorer taskbar present: %t)", taskbar != 0)
		}
		fmt.Printf("NOTE: notification area unavailable (Explorer taskbar present: %t); Close minimizes instead of hiding.\n", taskbar != 0)
	}
	setTimer.Call(s.window, 1, 250, 0)
	return nil
}

func run() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	selfTest := flag.Bool("self-test", false, "test all sounds without an audio device or keyboard hook")
	smoke := flag.Bool("smoke-test", false, "test the window, tray and keyboard listener, then exit")
	noAudio := flag.Bool("no-audio", false, "disable audio for the smoke test only")
	allowMissingTray := flag.Bool("allow-missing-tray", false, "allow a smoke test without a tray only when Explorer is absent")
	bankPath := flag.String("bank", filepath.Join(filepath.Dir(executable), "Keybed.soundbank"), "preloaded sound bank path")
	flag.Parse()
	if *noAudio && !*smoke {
		return fmt.Errorf("--no-audio is only supported with --smoke-test")
	}
	if *allowMissingTray && !*smoke {
		return fmt.Errorf("--allow-missing-tray is only supported with --smoke-test")
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
	s := &studio{settings: readSettings(settingsPath()), mixer: newMixer(bank), smoke: *smoke, allowMissingTray: *allowMissingTray}
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
	showWindow.Call(s.window, 5)
	if *smoke {
		setTimer.Call(s.window, 2, 1500, 0)
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
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
	if *smoke {
		if s.smokeError != nil {
			return s.smokeError
		}
		if s.trayAvailable {
			fmt.Println("PASS: native window, controls, tray icon, global keyboard hook and clean shutdown.")
		} else {
			fmt.Println("PASS: native window, controls, global keyboard hook and clean shutdown. Tray check unavailable: this desktop has no Explorer taskbar.")
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
