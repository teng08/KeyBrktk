//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

func (s *studio) creditsProcedure(window, message, wparam, lparam uintptr) uintptr {
	switch message {
	case 0x05: // Keep the read-only, scrollable text reachable after resizing.
		if s.creditsEdit != 0 {
			r := clientRect(window)
			moveWindow.Call(s.creditsEdit, 0, 0, uintptr(r.right), uintptr(r.bottom), 1)
		}
	case 0x138: // WM_CTLCOLORSTATIC: read-only EDIT controls use this notification.
		setTextColor.Call(wparam, studioText)
		setBGColor.Call(wparam, studioBackground)
		return s.backgroundBrush
	case 0x02:
		s.creditsWindow, s.creditsEdit = 0, 0
		return 0 // Closing credits must never quit the studio or keyboard listener.
	case 0x02e0:
		r := (*nativeRect)(unsafe.Pointer(lparam))
		setWindowPos.Call(window, 0, uintptr(r.left), uintptr(r.top), uintptr(r.right-r.left), uintptr(r.bottom-r.top), 0x14)
		return 0
	}
	result, _, _ := defWindowProc.Call(window, message, wparam, lparam)
	return result
}

func (s *studio) showCredits() error {
	if s.creditsWindow != 0 {
		showWindow.Call(s.creditsWindow, 9)
		setForegroundWindow.Call(s.creditsWindow)
		return nil
	}
	module, _, _ := getModuleHandle.Call(0)
	if !s.creditsRegistered {
		cursor, _, _ := loadCursor.Call(0, 32512)
		class := windowClass{procedure: syscall.NewCallback(s.creditsProcedure), instance: module, cursor: cursor, background: s.backgroundBrush, className: wide("Keybed.Credits")}
		if result, _, err := registerClass.Call(uintptr(unsafe.Pointer(&class))); result == 0 {
			return winError("register credits window", err)
		}
		s.creditsRegistered = true
	}
	work := monitorWork(s.window)
	w, h := min(s.px(720), work.w-16), min(s.px(540), work.h-16)
	window, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(wide("Keybed.Credits"))), uintptr(unsafe.Pointer(wide("Keybed · Sound credits and licenses"))), 0x02cf0000,
		uintptr(work.x+(work.w-w)/2), uintptr(work.y+(work.h-h)/2), uintptr(w), uintptr(h), s.window, 0, module, 0)
	if window == 0 {
		return winError("create credits window", err)
	}
	s.creditsWindow = window
	r := clientRect(window)
	text := strings.ReplaceAll(strings.ReplaceAll(appCredits(), "\r\n", "\n"), "\n", "\r\n")
	s.creditsEdit, _, err = createWindow.Call(0, uintptr(unsafe.Pointer(wide("EDIT"))), uintptr(unsafe.Pointer(wide(text))), 0x50210844,
		0, 0, uintptr(r.right), uintptr(r.bottom), window, 1, module, 0) // multiline, read-only, tab-stop, vertical scrolling
	if s.creditsEdit == 0 {
		destroyWindow.Call(window)
		return winError("create credits text", err)
	}
	sendMessage.Call(s.creditsEdit, 0x30, s.fonts[0], 1)
	showWindow.Call(window, 5)
	setForegroundWindow.Call(window)
	setFocus.Call(s.creditsEdit)
	return nil
}
