//go:build windows

package main

import (
	"testing"
	"unsafe"
)

func TestNativeLayouts(t *testing.T) {
	pointerSize := unsafe.Sizeof(uintptr(0))
	expected := map[string][2]uintptr{
		// Values are the documented Win32 and Win64 structure sizes.
		"WNDCLASSW":       {40, 72},
		"MSG":             {32, 48},
		"NOTIFYICONDATAW": {956, 976},
		"WAVEHDR":         {32, 48},
		"KBDLLHOOKSTRUCT": {20, 24},
		"DRAWITEMSTRUCT":  {48, 64},
		"PAINTSTRUCT":     {64, 72},
		"SCROLLINFO":      {28, 28},
		"MONITORINFO":     {40, 40},
	}
	actual := map[string]uintptr{
		"WNDCLASSW":       unsafe.Sizeof(windowClass{}),
		"MSG":             unsafe.Sizeof(windowMessage{}),
		"NOTIFYICONDATAW": unsafe.Sizeof(iconData{}),
		"WAVEHDR":         unsafe.Sizeof(waveHeader{}),
		"KBDLLHOOKSTRUCT": unsafe.Sizeof(keyboardInfo{}),
		"DRAWITEMSTRUCT":  unsafe.Sizeof(drawItem{}),
		"PAINTSTRUCT":     unsafe.Sizeof(paintInfo{}),
		"SCROLLINFO":      unsafe.Sizeof(scrollInfo{}),
		"MONITORINFO":     unsafe.Sizeof(monitorInfo{}),
	}
	index := 1
	if pointerSize == 4 {
		index = 0
	} else if pointerSize != 8 {
		t.Fatalf("unsupported pointer size %d", pointerSize)
	}
	for name, got := range actual {
		want := expected[name][index]
		if got != want {
			t.Fatalf("%s size %d, want %d for %d-bit Windows", name, got, want, pointerSize*8)
		}
	}
	if classifyKey(32) != 1 || classifyKey(13) != 2 || classifyKey(8) != 3 || classifyKey(46) != 3 || classifyKey(65) != 0 {
		t.Fatal("Windows key classification failed")
	}
}
