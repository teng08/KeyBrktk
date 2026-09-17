//go:build windows

package main

import (
	"testing"
	"unsafe"
)

func TestNative64BitLayouts(t *testing.T) {
	for name, layout := range map[string]struct{ got, want uintptr }{
		"WNDCLASSW":       {unsafe.Sizeof(windowClass{}), 72},
		"MSG":             {unsafe.Sizeof(windowMessage{}), 48},
		"NOTIFYICONDATAW": {unsafe.Sizeof(iconData{}), 976},
		"WAVEHDR":         {unsafe.Sizeof(waveHeader{}), 48},
		"KBDLLHOOKSTRUCT": {unsafe.Sizeof(keyboardInfo{}), 24},
		"DRAWITEMSTRUCT":  {unsafe.Sizeof(drawItem{}), 64},
		"PAINTSTRUCT":     {unsafe.Sizeof(paintInfo{}), 72},
		"SCROLLINFO":      {unsafe.Sizeof(scrollInfo{}), 28},
		"MONITORINFO":     {unsafe.Sizeof(monitorInfo{}), 40},
	} {
		if layout.got != layout.want {
			t.Fatalf("%s size %d, want %d", name, layout.got, layout.want)
		}
	}
	if classifyKey(32) != 1 || classifyKey(13) != 2 || classifyKey(8) != 3 || classifyKey(46) != 3 || classifyKey(65) != 0 {
		t.Fatal("Windows key classification failed")
	}
}
