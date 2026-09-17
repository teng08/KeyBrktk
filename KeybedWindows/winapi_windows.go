//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	shell32             = syscall.NewLazyDLL("shell32.dll")
	winmm               = syscall.NewLazyDLL("winmm.dll")
	registerClass       = user32.NewProc("RegisterClassW")
	registerMessage     = user32.NewProc("RegisterWindowMessageW")
	createWindow        = user32.NewProc("CreateWindowExW")
	defWindowProc       = user32.NewProc("DefWindowProcW")
	getMessage          = user32.NewProc("GetMessageW")
	peekMessage         = user32.NewProc("PeekMessageW")
	translateMessage    = user32.NewProc("TranslateMessage")
	dispatchMessage     = user32.NewProc("DispatchMessageW")
	showWindow          = user32.NewProc("ShowWindow")
	isWindowVisible     = user32.NewProc("IsWindowVisible")
	isIconic            = user32.NewProc("IsIconic")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	findWindow          = user32.NewProc("FindWindowW")
	destroyWindow       = user32.NewProc("DestroyWindow")
	postQuitMessage     = user32.NewProc("PostQuitMessage")
	postThreadMessage   = user32.NewProc("PostThreadMessageW")
	sendMessage         = user32.NewProc("SendMessageW")
	setWindowText       = user32.NewProc("SetWindowTextW")
	setTimer            = user32.NewProc("SetTimer")
	loadIcon            = user32.NewProc("LoadIconW")
	loadCursor          = user32.NewProc("LoadCursorW")
	messageBox          = user32.NewProc("MessageBoxW")
	createPopupMenu     = user32.NewProc("CreatePopupMenu")
	appendMenu          = user32.NewProc("AppendMenuW")
	trackPopupMenu      = user32.NewProc("TrackPopupMenu")
	destroyMenu         = user32.NewProc("DestroyMenu")
	getCursorPos        = user32.NewProc("GetCursorPos")
	setWindowsHook      = user32.NewProc("SetWindowsHookExW")
	callNextHook        = user32.NewProc("CallNextHookEx")
	unhookWindowsHook   = user32.NewProc("UnhookWindowsHookEx")
	getModuleHandle     = kernel32.NewProc("GetModuleHandleW")
	getCurrentThreadID  = kernel32.NewProc("GetCurrentThreadId")
	createMutex         = kernel32.NewProc("CreateMutexW")
	closeHandle         = kernel32.NewProc("CloseHandle")
	virtualAlloc        = kernel32.NewProc("VirtualAlloc")
	virtualFree         = kernel32.NewProc("VirtualFree")
	createEvent         = kernel32.NewProc("CreateEventW")
	waitForSingleObject = kernel32.NewProc("WaitForSingleObject")
	notifyIcon          = shell32.NewProc("Shell_NotifyIconW")
	initializeCOM       = syscall.NewLazyDLL("ole32.dll").NewProc("CoInitializeEx")
	uninitializeCOM     = syscall.NewLazyDLL("ole32.dll").NewProc("CoUninitialize")
	waveOutOpen         = winmm.NewProc("waveOutOpen")
	waveOutPrepare      = winmm.NewProc("waveOutPrepareHeader")
	waveOutWrite        = winmm.NewProc("waveOutWrite")
	waveOutUnprepare    = winmm.NewProc("waveOutUnprepareHeader")
	waveOutReset        = winmm.NewProc("waveOutReset")
	waveOutClose        = winmm.NewProc("waveOutClose")
)

type windowClass struct {
	style                              uint32
	procedure                          uintptr
	classExtra, windowExtra            int32
	instance, icon, cursor, background uintptr
	menuName, className                *uint16
}

type windowMessage struct {
	window         uintptr
	message        uint32
	wparam, lparam uintptr
	time           uint32
	point          [2]int32
	private        uint32
}

type iconData struct {
	size                uint32
	window              uintptr
	id, flags, callback uint32
	icon                uintptr
	tip                 [128]uint16
	state, stateMask    uint32
	info                [256]uint16
	version             uint32
	infoTitle           [64]uint16
	infoFlags           uint32
	guid                [16]byte
	balloonIcon         uintptr
}

func wide(text string) *uint16 { return syscall.StringToUTF16Ptr(text) }
func setText(window uintptr, text string) {
	setWindowText.Call(window, uintptr(unsafe.Pointer(wide(text))))
}
func showError(text string) {
	messageBox.Call(0, uintptr(unsafe.Pointer(wide(text))), uintptr(unsafe.Pointer(wide("Keybed"))), 0x10)
}
func winError(operation string, err error) error { return fmt.Errorf("%s: %w", operation, err) }
