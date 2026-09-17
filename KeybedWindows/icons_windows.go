//go:build windows

package main

import "unsafe"

// Native vector glyphs mirror the Mac's sound identities without font fallback.
func (s *studio) soundIcon(dc uintptr, index int, color uintptr) {
	pen, _, _ := createPen.Call(0, uintptr(max(1, s.px(1))), color)
	brush, _, _ := createBrush.Call(color)
	oldPen, _, _ := selectObject.Call(dc, pen)
	oldBrush, _, _ := selectObject.Call(dc, brush)
	defer func() {
		selectObject.Call(dc, oldPen)
		selectObject.Call(dc, oldBrush)
		deleteObject.Call(pen)
		deleteObject.Call(brush)
	}()
	line := func(x1, y1, x2, y2 int) {
		lineMove.Call(dc, uintptr(s.px(x1)), uintptr(s.px(y1)), 0)
		lineTo.Call(dc, uintptr(s.px(x2)), uintptr(s.px(y2)))
	}
	dot := func(x, y, w, h int) {
		gdi32.NewProc("Ellipse").Call(dc, uintptr(s.px(x)), uintptr(s.px(y)), uintptr(s.px(x+w)), uintptr(s.px(y+h)))
	}
	shape := func(points ...[2]int) {
		var native [8][2]int32
		for i, p := range points {
			native[i] = [2]int32{int32(s.px(p[0])), int32(s.px(p[1]))}
		}
		gdi32.NewProc("Polygon").Call(dc, uintptr(unsafe.Pointer(&native[0])), uintptr(len(points)))
	}
	switch index {
	case 0:
		null, _, _ := gdi32.NewProc("GetStockObject").Call(5)
		selectObject.Call(dc, null)
		roundRect.Call(dc, uintptr(s.px(15)), uintptr(s.px(14)), uintptr(s.px(37)), uintptr(s.px(29)), uintptr(s.px(3)), uintptr(s.px(3)))
		for row := 0; row < 2; row++ {
			for col := 0; col < 5; col++ {
				fill(dc, s.rectangle(18+col*3, 18+row*3, 1, 1), color)
			}
		}
		line(21, 25, 31, 25)
	case 1:
		for y := 15; y <= 23; y += 4 {
			line(15, y+2, 26, y+8)
			line(26, y+8, 37, y+2)
			if y == 15 {
				line(15, y+2, 26, y-4)
				line(26, y-4, 37, y+2)
			}
		}
	case 2:
		shape([2]int{28, 11}, [2]int{18, 24}, [2]int{25, 24}, [2]int{22, 34}, [2]int{35, 19}, [2]int{27, 19})
	case 3:
		for i, h := range []int{8, 17, 23, 12, 5} {
			fill(dc, s.rectangle(16+i*4, 23-h/2, 2, h), color)
		}
	case 4:
		for _, p := range [][2]int{{20, 14}, {29, 14}, {16, 21}, {25, 21}, {34, 21}, {20, 28}, {29, 28}} {
			dot(p[0], p[1], 5, 5)
		}
	case 5:
		s.text(dc, "abc", s.rectangle(14, 10, 30, 24), 4, color, false)
	case 6:
		shape([2]int{26, 12}, [2]int{17, 25}, [2]int{20, 31}, [2]int{32, 31}, [2]int{35, 25})
		dot(20, 22, 12, 12)
	case 7:
		rounded(dc, s.rectangle(15, 16, 25, 17), s.px(10), s.px(1), color, color)
		fill(dc, s.rectangle(20, 20, 2, 9), studioBackground)
		fill(dc, s.rectangle(17, 23, 8, 2), studioBackground)
		fill(dc, s.rectangle(31, 21, 2, 2), studioBackground)
		fill(dc, s.rectangle(34, 25, 2, 2), studioBackground)
	case 8:
		fill(dc, s.rectangle(27, 12, 2, 17), color)
		line(28, 12, 35, 10)
		line(35, 10, 35, 25)
		dot(21, 24, 8, 6)
		dot(29, 20, 7, 6)
	case 9:
		shape([2]int{15, 20}, [2]int{20, 20}, [2]int{26, 14}, [2]int{26, 31}, [2]int{20, 26}, [2]int{15, 26})
		line(31, 18, 34, 22)
		line(34, 22, 31, 27)
		line(35, 14, 39, 22)
		line(39, 22, 35, 31)
	case 10:
		dot(22, 21, 11, 11)
		for _, p := range [][2]int{{15, 18}, {21, 12}, {29, 12}, {35, 18}} {
			dot(p[0], p[1], 5, 7)
		}
	case 11:
		dot(15, 12, 24, 24)
	case 12:
		shape([2]int{26, 11}, [2]int{38, 23}, [2]int{26, 35}, [2]int{14, 23})
	}
}
