//go:build windows

package main

import (
	"fmt"
	"time"
	"unsafe"
)

func (s *studio) checkStudio() error {
	if s.smokeError != nil {
		return s.smokeError
	}
	if !s.trayAvailable {
		taskbar, _, _ := findWindow.Call(uintptr(unsafe.Pointer(wide("Shell_TrayWnd"))), 0)
		if !s.allowTrayFallback {
			return fmt.Errorf("tray registration failed after retries (Explorer present: %t)", taskbar != 0)
		}
		fmt.Printf("NOTE: experimental Windows ARM64 tray registration failed (Explorer present: %t); checking minimize-to-taskbar fallback, not marking tray registration as passed.\n", taskbar != 0)
	}
	for index, name := range presetNames {
		if s.cards[index] == 0 {
			return fmt.Errorf("sound card missing: %s", name)
		}
		// Exercise the real button click, native notification routing and audio path.
		sendMessage.Call(s.cards[index], 0xf5, 0, 0)
		if s.settings.Preset != index || s.mixer.preset.Load() != int32(index) {
			return fmt.Errorf("sound card did not configure audio: %s", name)
		}
		layout := soundCardLayout(int(float64(clientRect(s.library).right) / s.scale))[index]
		s.libraryScroll = s.px(max(0, layout.y+layout.h-276))
		s.layoutLibrary()
		var rect nativeRect
		getWindowRect.Call(s.cards[index], uintptr(unsafe.Pointer(&rect)))
		var origin [2]int32
		user32.NewProc("ClientToScreen").Call(s.library, uintptr(unsafe.Pointer(&origin)))
		viewport := clientRect(s.library)
		if rect.left < origin[0] || rect.top < origin[1] || rect.right > origin[0]+viewport.right || rect.bottom > origin[1]+viewport.bottom {
			return fmt.Errorf("sound card cannot scroll completely into view: %s, rect=%+v viewport=%+v", name, rect, viewport)
		}
	}
	for index, name := range intensityNames {
		sendMessage.Call(s.modes[index], 0xf5, 0, 0)
		if s.settings.Intensity != index || s.mixer.intensity.Load() != int32(index) {
			return fmt.Errorf("intensity did not configure audio: %s", name)
		}
	}
	sendMessage.Call(s.volume, 0x405, 1, 61)
	sendMessage.Call(s.window, 0x114, 0, s.volume)
	if s.settings.Volume != 61 {
		return fmt.Errorf("volume did not configure audio")
	}
	sendMessage.Call(s.releases, 0xf1, 0, 0)
	sendMessage.Call(s.window, 0x111, idReleases, s.releases)
	if s.settings.Releases || s.mixer.releases.Load() {
		return fmt.Errorf("release switch did not configure audio")
	}
	wasMuted := s.settings.Muted
	sendMessage.Call(s.mute, 0xf5, 0, 0)
	if s.settings.Muted == wasMuted || s.mixer.muted.Load() != s.settings.Muted {
		return fmt.Errorf("mute did not configure audio")
	}
	// Overlay disabled/enabled, real activity, idle collapse, reset, no activation.
	s.settings.Overlay = false
	s.refreshControls()
	if visible, _, _ := isWindowVisible.Call(s.hud); visible != 0 {
		return fmt.Errorf("overlay off did not hide counter")
	}
	s.settings.Overlay = true
	s.refreshControls()
	s.counter.reset()
	s.previewTime = time.Time{}
	now := time.Now()
	s.counter.record(now)
	s.counter.record(now)
	focus, _, _ := getFocus.Call()
	for index := 0; index < 30; index++ {
		s.tickHUD(now)
	}
	if s.hudCount != 2 || s.hudHeight < 65 {
		return fmt.Errorf("typing did not expand/count in HUD")
	}
	for index := 0; index < 30; index++ {
		s.tickHUD(now.Add(3500 * time.Millisecond))
	}
	if s.hudCount != 0 || s.hudHeight > 37 {
		return fmt.Errorf("idle HUD did not reset and collapse")
	}
	if after, _, _ := getFocus.Call(); after != focus {
		return fmt.Errorf("HUD stole keyboard focus")
	}
	s.counter.record(time.Now())
	sendMessage.Call(s.window, 0x111, idReset, 0)
	if s.counter.snapshot(time.Now()) != 0 {
		return fmt.Errorf("reset button did not clear counter")
	}
	// Native scrollbar input, bottom reachability on small windows and DPI changes.
	s.libraryScroll = 0
	s.layoutLibrary()
	sendMessage.Call(s.library, 0x115, 1, 0)
	if s.libraryScroll <= 0 {
		return fmt.Errorf("library scrollbar did not move")
	}
	var old nativeRect
	getWindowRect.Call(s.window, uintptr(unsafe.Pointer(&old)))
	setWindowPos.Call(s.window, 0, uintptr(old.left), uintptr(old.top), uintptr(s.px(760)), uintptr(s.px(480)), 0x14)
	sendMessage.Call(s.window, 0x115, 7, 0)
	if s.rootScroll <= 0 {
		return fmt.Errorf("small-screen studio did not scroll")
	}
	for _, control := range s.controls {
		if control.rect.y == 651 {
			var rect nativeRect
			getWindowRect.Call(control.window, uintptr(unsafe.Pointer(&rect)))
			var origin [2]int32
			user32.NewProc("ClientToScreen").Call(s.window, uintptr(unsafe.Pointer(&origin)))
			if rect.top < origin[1] || rect.bottom > origin[1]+clientRect(s.window).bottom {
				return fmt.Errorf("small-screen footer controls are inaccessible")
			}
		}
	}
	setWindowPos.Call(s.window, 0, uintptr(old.left), uintptr(old.top), uintptr(old.right-old.left), uintptr(old.bottom-old.top), 0x14)
	s.rootScroll = 0
	s.libraryScroll = 0
	s.layoutStudio()
	// Process the same production layout/font path as a monitor DPI change.
	for _, dpi := range []uint32{144, 192, s.dpi} {
		proposed := nativeRect{old.left, old.top, old.left + int32(760*dpi/96), old.top + int32(714*dpi/96)}
		sendMessage.Call(s.window, 0x2e0, uintptr(dpi)|uintptr(dpi)<<16, uintptr(unsafe.Pointer(&proposed)))
		if s.dpi != dpi || s.fonts[0] == 0 {
			return fmt.Errorf("DPI scaling did not update studio")
		}
	}
	setWindowPos.Call(s.window, 0, uintptr(old.left), uintptr(old.top), uintptr(old.right-old.left), uintptr(old.bottom-old.top), 0x14)
	s.rootScroll = 0
	s.libraryScroll = 0
	s.layoutStudio()
	registered := s.trayAvailable
	sendMessage.Call(s.window, 0x10, 0, 0)
	if registered {
		if visible, _, _ := isWindowVisible.Call(s.window); visible != 0 {
			return fmt.Errorf("close did not hide studio")
		}
	} else {
		if minimized, _, _ := isIconic.Call(s.window); minimized == 0 {
			return fmt.Errorf("no-tray close did not minimize")
		}
	}
	s.show()
	if visible, _, _ := isWindowVisible.Call(s.window); visible == 0 {
		return fmt.Errorf("studio did not reopen")
	}
	s.trayAvailable = false
	sendMessage.Call(s.window, 0x10, 0, 0)
	s.trayAvailable = registered
	if minimized, _, _ := isIconic.Call(s.window); minimized == 0 {
		return fmt.Errorf("no-tray fallback did not minimize")
	}
	s.show()
	if minimized, _, _ := isIconic.Call(s.window); minimized != 0 {
		return fmt.Errorf("no-tray fallback did not reopen")
	}
	if s.settings.Muted {
		s.toggleMute()
	}
	s.settings.Releases = true
	s.refreshControls()
	s.counter.record(time.Now())
	s.tickHUD(time.Now())
	if s.output == nil {
		setText(s.status, "Global keyboard listener active · audio disabled for CI")
	} else {
		setText(s.status, s.output.status.Load().(string))
	}
	if s.screenshotPath != "" {
		if err := captureStudio(s.window, s.screenshotPath); err != nil {
			return err
		}
	}
	return nil
}
