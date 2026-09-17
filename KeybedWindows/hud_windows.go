//go:build windows

package main

import (
	"fmt"
	"math"
	"syscall"
	"time"
	"unsafe"
)

// A layered, non-activating tool window never steals focus or mouse clicks.
func (s *studio) createHUD(module, cursor uintptr) error {
	s.hudScale = s.scale
	class := windowClass{procedure: syscall.NewCallback(s.hudProcedure), instance: module, cursor: cursor, className: wide("Keybed.CursorCounter")}
	result, _, err := registerClass.Call(uintptr(unsafe.Pointer(&class)))
	if result == 0 {
		return winError("register floating counter", err)
	}
	s.hud, _, err = createWindow.Call(0x080800a8, uintptr(unsafe.Pointer(class.className)), uintptr(unsafe.Pointer(wide("Keybed floating typing counter"))), 0x80000000,
		0, 0, uintptr(s.px(218)), uintptr(s.px(66)), 0, 0, module, 0)
	if s.hud == 0 {
		return winError("create floating counter", err)
	}
	setLayeredAttributes.Call(s.hud, 0x030201, 245, 3)
	return s.setHUDFonts()
}

func (s *studio) hp(value int) int { return int(math.Round(float64(value) * s.hudScale)) }
func (s *studio) hudRect(x, y, w, h int) nativeRect {
	return nativeRect{int32(s.hp(x)), int32(s.hp(y)), int32(s.hp(x + w)), int32(s.hp(y + h))}
}
func (s *studio) setHUDFonts() error {
	for index, size := range [...]int{11, 16} {
		weight := 400
		if index == 1 {
			weight = 600
		}
		font, _, err := createFont.Call(uintptr(-s.hp(size)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wide("Segoe UI"))))
		if font == 0 {
			return winError("create floating counter font", err)
		}
		if s.hudFonts[index] != 0 {
			deleteObject.Call(s.hudFonts[index])
		}
		s.hudFonts[index] = font
	}
	return nil
}
func (s *studio) hudText(dc uintptr, text string, rect nativeRect, font int, color uintptr) {
	drawNativeText(dc, text, rect, s.hudFonts[font], color, false)
}

func (s *studio) tickHUD(now time.Time) {
	if s.hud == 0 {
		return
	}
	if !s.settings.Overlay {
		showWindow.Call(s.hud, 0)
		return
	}
	count, last := s.counter.activity(now)
	age := now.Sub(last)
	if s.previewTime.After(last) {
		age = now.Sub(s.previewTime)
	}
	target := 0.0
	if last.IsZero() && s.previewTime.IsZero() || age >= 3*time.Second {
		target = 1
	}
	s.hudCollapse += (target - s.hudCollapse) * .28
	if math.Abs(target-s.hudCollapse) < .002 {
		s.hudCollapse = target
	}
	height := int(math.Round(66 - s.hudCollapse*30))
	var point [2]int32
	getCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	packed := uintptr(uint64(uint32(point[0])) | uint64(uint32(point[1]))<<32)
	monitor, _, _ := monitorFromPoint.Call(packed, 2)
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	getMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info)))
	r := info.work
	x, y := floatingOrigin(int(point[0]), int(point[1]), layoutRect{int(r.left), int(r.top), int(r.right - r.left), int(r.bottom - r.top)}, s.hp(218), s.hp(height))
	setWindowPos.Call(s.hud, ^uintptr(0), uintptr(x), uintptr(y), uintptr(s.hp(218)), uintptr(s.hp(height)), 0x50)
	if s.hudHeight != height || s.hudCount != count || s.hudLast != last || age < 800*time.Millisecond {
		s.hudHeight, s.hudCount, s.hudLast = height, count, last
		repaint(s.hud)
	}
}

func (s *studio) hudProcedure(window, message, wparam, lparam uintptr) uintptr {
	switch message {
	case 0x2e0:
		s.hudScale = float64(wparam&0xffff) / 96
		if err := s.setHUDFonts(); err != nil {
			s.settings.Overlay = false
			s.smokeError = err
			showWindow.Call(window, 0)
		}
		repaint(window)
		return 0
	case 0x84:
		return ^uintptr(0) // HTTRANSPARENT
	case 0x21:
		return 3 // MA_NOACTIVATE
	case 0x14:
		return 1
	case 0xf:
		var paint paintInfo
		dc, _, _ := beginPaint.Call(window, uintptr(unsafe.Pointer(&paint)))
		r := clientRect(window)
		fill(dc, r, 0x030201)
		r.left += int32(s.hp(2))
		r.top += int32(s.hp(2))
		r.right -= int32(s.hp(2))
		r.bottom -= int32(s.hp(2))
		accent := presetColor(s.settings.Preset)
		rounded(dc, r, s.hp(30), s.hp(1), rgb(.075, .085, .105), accent)
		height := int(float64(r.bottom) / s.hudScale)
		if height > 40 {
			s.hudText(dc, fmt.Sprintf("%d keys", s.hudCount), s.hudRect(57, 10, 152, 24), 1, studioText)
		}
		y := 36
		if height <= 40 {
			y = 10
		}
		label := presetNames[s.settings.Preset]
		if s.settings.Muted {
			label = "Muted · " + label
		}
		s.hudText(dc, label, s.hudRect(57, y, 152, 18), 0, accent)
		age := time.Since(s.hudLast)
		if s.previewTime.After(s.hudLast) {
			age = time.Since(s.previewTime)
		}
		pulse := 0.0
		if age >= 0 && age < 800*time.Millisecond {
			pulse = math.Exp(-age.Seconds() * 8)
		}
		center := height / 2
		for index := 0; index < 5; index++ {
			bar := 5 + int(pulse*(9+math.Abs(math.Sin(float64(index)*1.4+float64(time.Now().UnixMilli())/100))*11))
			fill(dc, s.hudRect(19+index*4, center-bar/2, 2, bar), accent)
		}
		endPaint.Call(window, uintptr(unsafe.Pointer(&paint)))
		return 0
	}
	result, _, _ := defWindowProc.Call(window, message, wparam, lparam)
	return result
}
