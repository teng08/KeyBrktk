//go:build windows

package main

import (
	"fmt"
	"math"
	"syscall"
	"time"
	"unsafe"
)

const (
	idCardBase   = 200
	idModeBase   = 300
	idOverlay    = 110
	idReset      = 111
	idStartup    = 112
	studioHeight = 714
)

type placedControl struct {
	window uintptr
	rect   layoutRect
	font   int
}

type studioUI struct {
	library, overlay, startup, selectedSound, hud uintptr
	cards                                         [len(presetNames)]uintptr
	modes                                         [len(intensityNames)]uintptr
	controls                                      []placedControl
	fonts                                         [5]uintptr
	backgroundBrush                               uintptr
	dpi                                           uint32
	scale                                         float64
	rootScroll, libraryScroll                     int
	layingOut, startupEnabled                     bool
	hudHeight, hudCount                           int
	hudLast                                       time.Time
	previewTime                                   time.Time
	hudCollapse                                   float64
	hudScale                                      float64
	hudFonts                                      [2]uintptr
	screenshotPath                                string
}

func rgb(r, g, b float64) uintptr {
	return uintptr(math.Round(r*255)) | uintptr(math.Round(g*255))<<8 | uintptr(math.Round(b*255))<<16
}

var studioBackground = rgb(.065, .075, .095)
var studioText = rgb(.96, .96, .97)
var studioSecondary = rgb(.66, .66, .68)

func presetColor(index int) uintptr {
	p := presetLooks[index]
	return rgb(p.r, p.g, p.b)
}
func (s *studio) px(value int) int { return int(math.Round(float64(value) * s.scale)) }
func (s *studio) rectangle(x, y, w, h int) nativeRect {
	return nativeRect{int32(s.px(x)), int32(s.px(y)), int32(s.px(x + w)), int32(s.px(y + h))}
}
func clientRect(window uintptr) nativeRect {
	var r nativeRect
	getClientRect.Call(window, uintptr(unsafe.Pointer(&r)))
	return r
}
func repaint(window uintptr) {
	if window != 0 {
		invalidateRect.Call(window, 0, 1)
	}
}
func fill(dc uintptr, rect nativeRect, color uintptr) {
	brush, _, _ := createBrush.Call(color)
	fillRect.Call(dc, uintptr(unsafe.Pointer(&rect)), brush)
	deleteObject.Call(brush)
}
func rounded(dc uintptr, rect nativeRect, radius, stroke int, color, border uintptr) {
	brush, _, _ := createBrush.Call(color)
	pen, _, _ := createPen.Call(0, uintptr(max(1, stroke)), border)
	oldBrush, _, _ := selectObject.Call(dc, brush)
	oldPen, _, _ := selectObject.Call(dc, pen)
	roundRect.Call(dc, uintptr(rect.left), uintptr(rect.top), uintptr(rect.right), uintptr(rect.bottom), uintptr(radius), uintptr(radius))
	selectObject.Call(dc, oldPen)
	selectObject.Call(dc, oldBrush)
	deleteObject.Call(pen)
	deleteObject.Call(brush)
}
func (s *studio) text(dc uintptr, text string, rect nativeRect, font int, color uintptr, centered bool) {
	drawNativeText(dc, text, rect, s.fonts[font], color, centered)
}
func drawNativeText(dc uintptr, text string, rect nativeRect, font, color uintptr, centered bool) {
	old, _, _ := selectObject.Call(dc, font)
	setBGMode.Call(dc, 1)
	setTextColor.Call(dc, color)
	flags := uintptr(0x20 | 0x800 | 0x8000) // single line, no '&' mnemonic, ellipsis
	if centered {
		flags |= 1 | 4
	}
	drawText.Call(dc, uintptr(unsafe.Pointer(wide(text))), ^uintptr(0), uintptr(unsafe.Pointer(&rect)), flags)
	selectObject.Call(dc, old)
}

func (s *studio) setFonts() error {
	var next [5]uintptr
	for index, size := range [...]int{11, 13, 10, 28, 16} {
		weight := 400
		if index == 1 || index >= 3 {
			weight = 600
		}
		font, _, err := createFont.Call(uintptr(-s.px(size)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0,
			uintptr(unsafe.Pointer(wide("Segoe UI"))))
		if font == 0 {
			for _, made := range next {
				if made != 0 {
					deleteObject.Call(made)
				}
			}
			return winError("create studio font", err)
		}
		next[index] = font
	}
	old := s.fonts
	s.fonts = next
	for _, control := range s.controls {
		sendMessage.Call(control.window, 0x30, s.fonts[control.font], 1)
	}
	for _, font := range old {
		if font != 0 {
			deleteObject.Call(font)
		}
	}
	return nil
}

func (s *studio) uiControl(parent uintptr, class, text string, style uintptr, r layoutRect, id uintptr, font int) (uintptr, error) {
	module, _, _ := getModuleHandle.Call(0)
	extended := uintptr(0)
	if class == "Keybed.SoundLibrary" {
		extended = 0x10000
	}
	if class == "BUTTON" {
		style |= 0x4000
	} // notify focus so scrolled controls remain reachable by keyboard
	window, _, err := createWindow.Call(extended, uintptr(unsafe.Pointer(wide(class))), uintptr(unsafe.Pointer(wide(text))), style|0x50000000,
		0, 0, 1, 1, parent, id, module, 0)
	if window == 0 {
		return 0, winError("create "+class, err)
	}
	sendMessage.Call(window, 0x30, s.fonts[font], 1)
	if parent == s.window {
		s.controls = append(s.controls, placedControl{window, r, font})
	}
	if class == "BUTTON" || class == "msctls_trackbar32" {
		setWindowTheme.Call(window, uintptr(unsafe.Pointer(wide(""))), uintptr(unsafe.Pointer(wide(""))))
	}
	return window, nil
}

func (s *studio) createStudioUI() error {
	s.dpi = 96
	if getSystemDPI.Find() == nil {
		value, _, _ := getSystemDPI.Call()
		if value != 0 {
			s.dpi = uint32(value)
		}
	}
	s.scale = float64(s.dpi) / 96
	if err := s.setFonts(); err != nil {
		return err
	}
	s.backgroundBrush, _, _ = createBrush.Call(studioBackground)
	module, _, _ := getModuleHandle.Call(0)
	icon, _, _ := loadIcon.Call(0, 32512)
	cursor, _, _ := loadCursor.Call(0, 32512)
	class := windowClass{procedure: syscall.NewCallback(s.procedure), instance: module, icon: icon, cursor: cursor, background: s.backgroundBrush, className: wide(windowClassName)}
	result, _, err := registerClass.Call(uintptr(unsafe.Pointer(&class)))
	if result == 0 {
		s.disposeUI()
		return winError("register studio window", err)
	}
	style := uintptr(0x02cf0000 | 0x00200000) // resizable, clip children, vertical scrolling on short screens
	outer := nativeRect{right: int32(s.px(760)), bottom: int32(s.px(studioHeight))}
	adjustWindowForDPI.Call(uintptr(unsafe.Pointer(&outer)), style, 0, 0, uintptr(s.dpi))
	work := monitorWork(0)
	w, h := min(int(outer.right-outer.left), work.w-16), min(int(outer.bottom-outer.top), work.h-16)
	s.window, _, err = createWindow.Call(0, uintptr(unsafe.Pointer(class.className)), uintptr(unsafe.Pointer(wide("Keybed · Sound Studio"))), style,
		uintptr(work.x+(work.w-w)/2), uintptr(work.y+(work.h-h)/2), uintptr(w), uintptr(h), 0, 0, module, 0)
	if s.window == 0 {
		s.disposeUI()
		return winError("create studio window", err)
	}
	if dwm := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute"); dwm.Find() == nil {
		dark := uint32(1)
		result, _, _ := dwm.Call(s.window, 20, uintptr(unsafe.Pointer(&dark)), 4)
		if int32(result) < 0 {
			dwm.Call(s.window, 19, uintptr(unsafe.Pointer(&dark)), 4)
		}
	}
	if getDPI.Find() == nil {
		dpi, _, _ := getDPI.Call(s.window)
		if dpi != 0 {
			s.dpi = uint32(dpi)
			s.scale = float64(dpi) / 96
			if err := s.setFonts(); err != nil {
				return err
			}
		}
	}
	labels := []struct {
		text   string
		r      layoutRect
		font   int
		target *uintptr
	}{
		{"K E Y B E D   /   S O U N D   S T U D I O", layoutRect{28, 22, 430, 16}, 2, nil},
		{"Find your typing rhythm.", layoutRect{28, 43, 540, 38}, 3, nil},
		{fmt.Sprintf("%d sounds. Instant feedback. A little joy in every key.", len(presetNames)), layoutRect{28, 85, 535, 20}, 0, nil},
		{"3-SECOND COUNT", layoutRect{581, 27, 150, 16}, 2, nil},
		{"0", layoutRect{581, 46, 150, 38}, 3, &s.count},
		{"SOUND LIBRARY · SCROLL", layoutRect{28, 116, 145, 17}, 2, nil},
		{"", layoutRect{175, 113, 280, 22}, 0, &s.selectedSound},
		{"INTENSITY", layoutRect{28, 443, 96, 18}, 2, nil},
		{"", layoutRect{449, 443, 102, 18}, 0, &s.volumeLabel},
		{"Count resets every 3 seconds. The floating counter collapses when you pause.", layoutRect{28, 475, 704, 20}, 0, nil},
		{"Starting…", layoutRect{28, 584, 704, 22}, 1, &s.status},
		{"Keyboard sounds work across apps. Secure desktops and protected apps may block access.", layoutRect{28, 610, 704, 31}, 0, nil},
		{"Close this window to keep sounds and the counter running in your tray or taskbar.", layoutRect{28, 690, 704, 17}, 2, nil},
	}
	for _, label := range labels {
		handle, err := s.uiControl(s.window, "STATIC", label.text, 0, label.r, 0, label.font)
		if err != nil {
			return err
		}
		if label.target != nil {
			*label.target = handle
		}
	}
	tryField, err := s.uiControl(s.window, "EDIT", "", 0x00810080, layoutRect{462, 108, 270, 25}, 0, 0)
	if err != nil {
		return err
	}
	sendMessage.Call(tryField, 0x1501, 1, uintptr(unsafe.Pointer(wide("Type here to try your sound…"))))
	libClass := windowClass{procedure: syscall.NewCallback(s.libraryProcedure), instance: module, cursor: cursor, background: s.backgroundBrush, className: wide("Keybed.SoundLibrary")}
	result, _, err = registerClass.Call(uintptr(unsafe.Pointer(&libClass)))
	if result == 0 {
		return winError("register sound library", err)
	}
	s.library, err = s.uiControl(s.window, "Keybed.SoundLibrary", "Sound library", 0x02210000, layoutRect{28, 142, 704, 276}, 0, 0)
	if err != nil {
		return err
	}
	for index, name := range presetNames {
		s.cards[index], err = s.uiControl(s.library, "BUTTON", name+". "+presetLooks[index].detail, 0x1000b, layoutRect{}, idCardBase+uintptr(index), 1)
		if err != nil {
			return err
		}
	}
	for index, name := range intensityNames {
		s.modes[index], err = s.uiControl(s.window, "BUTTON", name, 0x1000b, layoutRect{127 + index*98, 436, 98, 28}, idModeBase+uintptr(index), 0)
		if err != nil {
			return err
		}
	}
	buttons := []struct {
		text   string
		r      layoutRect
		id     uintptr
		check  bool
		target *uintptr
	}{
		{"Play key release sounds", layoutRect{28, 507, 310, 24}, idReleases, true, &s.releases},
		{"Start automatically when I log in", layoutRect{380, 507, 350, 24}, idStartup, true, &s.startup},
		{"Floating counter follows my mouse", layoutRect{28, 541, 350, 24}, idOverlay, true, &s.overlay},
		{"Reset count", layoutRect{618, 539, 114, 28}, idReset, false, nil},
		{"▶  Preview sound", layoutRect{28, 651, 146, 30}, idPreview, false, nil},
		{"Mute", layoutRect{179, 651, 89, 30}, idMute, false, &s.mute},
		{"Quit Keybed", layoutRect{618, 651, 114, 30}, idQuit, false, nil},
	}
	for _, button := range buttons {
		style := uintptr(0x1000b)
		if button.check {
			style = 0x10003
		}
		handle, err := s.uiControl(s.window, "BUTTON", button.text, style, button.r, button.id, 0)
		if err != nil {
			return err
		}
		if button.target != nil {
			*button.target = handle
		}
	}
	s.volume, err = s.uiControl(s.window, "msctls_trackbar32", "Volume", 0x10010, layoutRect{548, 436, 183, 28}, idVolume, 0)
	if err != nil {
		return err
	}
	sendMessage.Call(s.volume, 0x406, 1, 100<<16)
	s.startupEnabled = loginEnabled()
	if err := s.createHUD(module, cursor); err != nil {
		return err
	}
	s.layoutStudio()
	s.refreshControls()
	s.icon = iconData{size: uint32(unsafe.Sizeof(iconData{})), window: s.window, id: 1, flags: 7, callback: trayMessage, icon: icon}
	copy(s.icon.tip[:], syscall.StringToUTF16("Keybed · keyboard sounds"))
	if !s.addTray() {
		fmt.Println("NOTE: notification area unavailable; Close minimizes instead of hiding.")
	}
	setTimer.Call(s.window, 1, 250, 0)
	setTimer.Call(s.window, 3, 33, 0)
	return nil
}

func monitorWork(window uintptr) layoutRect {
	monitor, _, _ := monitorFromWindow.Call(window, 2)
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	result, _, _ := getMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info)))
	if result == 0 {
		return layoutRect{0, 0, 1024, 768}
	}
	r := info.work
	return layoutRect{int(r.left), int(r.top), int(r.right - r.left), int(r.bottom - r.top)}
}
func updateScroll(window uintptr, docHeight, page, offset int) int {
	offset = clamp(offset, 0, max(0, docHeight-page))
	info := scrollInfo{size: uint32(unsafe.Sizeof(scrollInfo{})), mask: 7, max: int32(docHeight - 1), page: uint32(page), position: int32(offset)}
	setScrollInfo.Call(window, 1, uintptr(unsafe.Pointer(&info)), 1)
	return offset
}
func scrollCommand(window uintptr, command uintptr, current, page, line int) int {
	switch command & 0xffff {
	case 0:
		return current - line
	case 1:
		return current + line
	case 2:
		return current - page
	case 3:
		return current + page
	case 4, 5:
		info := scrollInfo{size: uint32(unsafe.Sizeof(scrollInfo{})), mask: 0x10}
		getScrollInfo.Call(window, 1, uintptr(unsafe.Pointer(&info)))
		return int(info.track)
	case 6:
		return 0
	case 7:
		return 1 << 20
	}
	return current
}

func (s *studio) layoutStudio() {
	if s.window == 0 || s.scale == 0 || s.layingOut {
		return
	}
	s.layingOut = true
	defer func() { s.layingOut = false }()
	r := clientRect(s.window)
	s.rootScroll = updateScroll(s.window, s.px(studioHeight), int(r.bottom), s.rootScroll)
	r = clientRect(s.window)
	for _, control := range s.controls {
		p := control.rect
		x := int(math.Round(float64(p.x) * float64(r.right) / 760))
		right := int(math.Round(float64(p.x+p.w) * float64(r.right) / 760))
		moveWindow.Call(control.window, uintptr(x), uintptr(s.px(p.y)-s.rootScroll), uintptr(right-x), uintptr(s.px(p.h)), 1)
	}
	s.layoutLibrary()
	repaint(s.window)
}
func (s *studio) layoutLibrary() {
	if s.library == 0 || s.scale == 0 {
		return
	}
	r := clientRect(s.library)
	s.libraryScroll = updateScroll(s.library, s.px(342), int(r.bottom), s.libraryScroll)
	r = clientRect(s.library)
	width := int(math.Floor(float64(r.right) / s.scale))
	for index, p := range soundCardLayout(width) {
		if s.cards[index] != 0 {
			moveWindow.Call(s.cards[index], uintptr(s.px(p.x)), uintptr(s.px(p.y)-s.libraryScroll), uintptr(s.px(p.w)), uintptr(s.px(p.h)), 1)
		}
	}
	repaint(s.library)
}
func (s *studio) refreshControls() {
	if s.window == 0 || s.mute == 0 {
		return
	}
	label := "Mute"
	if s.settings.Muted {
		label = "Unmute"
	}
	setText(s.mute, label)
	setText(s.selectedSound, "Now playing · "+presetNames[s.settings.Preset])
	setText(s.volumeLabel, fmt.Sprintf("Volume  %d%%", s.settings.Volume))
	sendMessage.Call(s.volume, 0x405, 1, uintptr(s.settings.Volume))
	for handle, checked := range map[uintptr]bool{s.releases: s.settings.Releases, s.overlay: s.settings.Overlay, s.startup: s.startupEnabled} {
		value := uintptr(0)
		if checked {
			value = 1
		}
		sendMessage.Call(handle, 0xf1, value, 0)
	}
	for _, card := range s.cards {
		repaint(card)
	}
	for _, mode := range s.modes {
		repaint(mode)
	}
	repaint(s.selectedSound)
	repaint(s.hud)
	s.tickHUD(time.Now())
}

func (s *studio) libraryProcedure(window, message, wparam, lparam uintptr) uintptr {
	switch message {
	case 0x111:
		id, notification := int(wparam&0xffff), wparam>>16&0xffff
		if notification == 6 && id >= idCardBase && id < idCardBase+len(s.cards) {
			card := soundCardLayout(int(float64(clientRect(window).right) / s.scale))[id-idCardBase]
			top, bottom := s.px(card.y), s.px(card.y+card.h)
			page := int(clientRect(window).bottom)
			if top < s.libraryScroll {
				s.libraryScroll = top
			}
			if bottom > s.libraryScroll+page {
				s.libraryScroll = bottom - page
			}
			s.layoutLibrary()
		}
		result, _, _ := sendMessage.Call(s.window, message, wparam, lparam)
		return result
	case 0x2b:
		result, _, _ := sendMessage.Call(s.window, message, wparam, lparam)
		return result
	case 0x115:
		s.libraryScroll = scrollCommand(window, wparam, s.libraryScroll, int(clientRect(window).bottom), s.px(70))
		s.layoutLibrary()
		return 0
	case 0x20a:
		s.libraryScroll -= int(int16(wparam>>16)) * s.px(70) / 120
		s.layoutLibrary()
		return 0
	case 0x100:
		switch wparam {
		case 0x22:
			s.libraryScroll += s.px(210)
		case 0x21:
			s.libraryScroll -= s.px(210)
		case 0x24:
			s.libraryScroll = 0
		case 0x23:
			s.libraryScroll = s.px(342)
		}
		s.layoutLibrary()
		return 0
	}
	result, _, _ := defWindowProc.Call(window, message, wparam, lparam)
	return result
}

func (s *studio) uiMessage(window, message, wparam, lparam uintptr) (uintptr, bool) {
	switch message {
	case 0x05:
		s.layoutStudio()
		return 0, false
	case 0x115:
		s.rootScroll = scrollCommand(window, wparam, s.rootScroll, int(clientRect(window).bottom), s.px(40))
		s.layoutStudio()
		return 0, true
	case 0x20a:
		s.rootScroll -= int(int16(wparam>>16)) * s.px(40) / 120
		s.layoutStudio()
		return 0, true
	case 0x2e0:
		s.dpi = uint32(wparam & 0xffff)
		s.scale = float64(s.dpi) / 96
		if err := s.setFonts(); err != nil {
			s.smokeError = err
			destroyWindow.Call(window)
			return 0, true
		}
		r := (*nativeRect)(unsafe.Pointer(lparam))
		work := monitorWork(window)
		setWindowPos.Call(window, 0, uintptr(r.left), uintptr(r.top), uintptr(min(int(r.right-r.left), work.w-16)), uintptr(min(int(r.bottom-r.top), work.h-16)), 0x14)
		s.layoutStudio()
		return 0, true
	case 0x2b:
		s.drawControl((*drawItem)(unsafe.Pointer(lparam)))
		return 1, true
	case 0x138, 0x133, 0x135:
		setBGColor.Call(wparam, studioBackground)
		setTextColor.Call(wparam, studioSecondary)
		if lparam == s.count {
			setTextColor.Call(wparam, studioText)
		}
		if lparam == s.selectedSound || lparam == s.status {
			setTextColor.Call(wparam, presetColor(s.settings.Preset))
		}
		for _, c := range s.controls {
			if c.window == lparam && c.rect.y == 22 {
				setTextColor.Call(wparam, presetColor(0))
			}
			if c.window == lparam && c.rect.y == 43 {
				setTextColor.Call(wparam, studioText)
			}
		}
		return s.backgroundBrush, true
	case 0x14:
		fillRect.Call(wparam, uintptr(unsafe.Pointer(&nativeRect{right: clientRect(window).right, bottom: clientRect(window).bottom})), s.backgroundBrush)
		return 1, true
	}
	return 0, false
}

func (s *studio) drawControl(item *drawItem) {
	r, dc, id := item.rect, item.dc, int(item.controlID)
	fill(dc, r, studioBackground)
	if id >= idCardBase && id < idCardBase+len(presetNames) {
		index := id - idCardBase
		accent := presetColor(index)
		background, border := rgb(.12, .12, .13), rgb(.23, .23, .24)
		stroke := s.px(1)
		if index == s.settings.Preset {
			p := presetLooks[index]
			background = rgb(.09+p.r*.08, .10+p.g*.08, .12+p.b*.08)
			border = accent
			stroke = s.px(2)
		}
		if item.state&1 != 0 {
			background = rgb(.18, .19, .20)
		}
		rounded(dc, r, s.px(22), stroke, background, border)
		s.soundIcon(dc, index, accent)
		width := int(float64(r.right) / s.scale)
		s.text(dc, presetNames[index], s.rectangle(47, 11, width-84, 21), 1, studioText, false)
		s.text(dc, presetLooks[index].detail, s.rectangle(14, 36, width-28, 18), 0, studioSecondary, false)
		badge := "NEW"
		if index == 0 {
			badge = ""
		}
		if index == s.settings.Preset {
			badge = "●"
		}
		s.text(dc, badge, s.rectangle(width-36, 19, 30, 16), 2, accent, false)
	} else {
		color, border := rgb(.22, .23, .25), rgb(.29, .30, .32)
		if id >= idModeBase && id < idModeBase+len(intensityNames) {
			color = rgb(.12, .13, .15)
			if id-idModeBase == s.settings.Intensity {
				color = rgb(.34, .35, .37)
			}
		}
		if item.state&1 != 0 {
			color = rgb(.38, .39, .41)
		}
		rounded(dc, r, s.px(8), s.px(1), color, border)
		text := ""
		switch id {
		case idPreview:
			text = "▶  Preview sound"
		case idMute:
			text = "Mute"
			if s.settings.Muted {
				text = "Unmute"
			}
		case idReset:
			text = "Reset count"
		case idQuit:
			text = "Quit Keybed"
		}
		if id >= idModeBase && id < idModeBase+len(intensityNames) {
			text = intensityNames[id-idModeBase]
		}
		s.text(dc, text, r, 0, studioText, true)
	}
	if item.state&0x10 != 0 {
		focus := r
		focus.left += 3
		focus.top += 3
		focus.right -= 3
		focus.bottom -= 3
		drawFocusRect.Call(dc, uintptr(unsafe.Pointer(&focus)))
	}
}

func (s *studio) disposeUI() {
	if s.hud != 0 {
		destroyWindow.Call(s.hud)
		s.hud = 0
	}
	for index, font := range s.hudFonts {
		if font != 0 {
			deleteObject.Call(font)
			s.hudFonts[index] = 0
		}
	}
	for index, font := range s.fonts {
		if font != 0 {
			deleteObject.Call(font)
			s.fonts[index] = 0
		}
	}
	if s.backgroundBrush != 0 {
		deleteObject.Call(s.backgroundBrush)
		s.backgroundBrush = 0
	}
}
