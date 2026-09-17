//go:build windows

package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"unsafe"
)

type bitmapInfo struct {
	size                   uint32
	width, height          int32
	planes, bits           uint16
	compression, sizeImage uint32
	x, y                   int32
	used, important        uint32
}

// Capture only our own app for CI review, not the desktop or another app.
func captureStudio(window uintptr, path string) error {
	r := clientRect(window)
	width, height := int(r.right), int(r.bottom)
	if width <= 0 || height <= 0 || width > 8192 || height > 8192 {
		return fmt.Errorf("invalid studio snapshot size")
	}
	dc, _, _ := user32.NewProc("GetDC").Call(window)
	if dc == 0 {
		return fmt.Errorf("get studio snapshot DC")
	}
	defer user32.NewProc("ReleaseDC").Call(window, dc)
	copyDC, _, _ := gdi32.NewProc("CreateCompatibleDC").Call(dc)
	if copyDC == 0 {
		return fmt.Errorf("create studio snapshot DC")
	}
	defer gdi32.NewProc("DeleteDC").Call(copyDC)
	bitmap, _, _ := gdi32.NewProc("CreateCompatibleBitmap").Call(dc, uintptr(width), uintptr(height))
	if bitmap == 0 {
		return fmt.Errorf("create studio snapshot bitmap")
	}
	defer deleteObject.Call(bitmap)
	old, _, _ := selectObject.Call(copyDC, bitmap)
	// WM_PRINTCLIENT paints native children, including owner-drawn sound cards.
	printed, _, _ := user32.NewProc("PrintWindow").Call(window, copyDC, 1)
	if printed == 0 {
		return fmt.Errorf("render studio snapshot")
	}
	selectObject.Call(copyDC, old)
	data := make([]byte, width*height*4)
	info := bitmapInfo{size: 40, width: int32(width), height: -int32(height), planes: 1, bits: 32}
	result, _, _ := gdi32.NewProc("GetDIBits").Call(dc, bitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&info)), 0)
	if int(result) != height {
		return fmt.Errorf("read studio snapshot bitmap")
	}
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	dark, background, textPixels := 0, 0, 0
	for offset := 0; offset < len(data); offset += 4 {
		output.Pix[offset], output.Pix[offset+1], output.Pix[offset+2], output.Pix[offset+3] = data[offset+2], data[offset+1], data[offset], 255
		if data[offset] < 70 && data[offset+1] < 70 && data[offset+2] < 70 {
			dark++
		}
		if data[offset] == 24 && data[offset+1] == 19 && data[offset+2] == 17 || data[offset] == 27 && data[offset+1] == 22 && data[offset+2] == 19 {
			background++
		}
		if data[offset] > 180 && data[offset+1] > 180 && data[offset+2] > 180 {
			textPixels++
		}
	}
	if dark < width*height/2 || background < width*height/15 || textPixels < max(20, min(100, width*height/1000)) {
		return fmt.Errorf("studio snapshot did not render the dark UI")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(file, output)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
