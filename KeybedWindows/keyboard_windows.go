//go:build windows

package main

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

type keyboardInfo struct {
	key, scan, flags, time uint32
	extra                  uintptr
}
type keyboardListener struct {
	thread uintptr
	done   chan struct{}
}

func classifyKey(key uint32) int {
	switch key {
	case 32:
		return 1
	case 13:
		return 2
	case 8, 46:
		return 3
	default:
		return 0
	}
}

func startKeyboard(m *mixer, counter *typingCounter) (*keyboardListener, error) {
	listener := &keyboardListener{done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(listener.done)
		var pressed [256]bool
		callback := syscall.NewCallback(func(code, message, data uintptr) uintptr {
			if int32(code) == 0 {
				info := (*keyboardInfo)(unsafe.Pointer(data))
				// Ignore software-injected keys; never swallow or alter real input.
				if info.key < 256 && info.flags&0x10 == 0 {
					switch message {
					case 0x100, 0x104:
						pressed[info.key] = true
						counter.record(time.Now())
						m.enqueue(keyEvent{kind: classifyKey(info.key)})
					case 0x101, 0x105:
						if pressed[info.key] {
							m.enqueue(keyEvent{kind: classifyKey(info.key), release: true})
							pressed[info.key] = false
						}
					}
				}
			}
			next, _, _ := callNextHook.Call(0, code, message, data)
			return next
		})
		listener.thread, _, _ = getCurrentThreadID.Call()
		var message windowMessage
		peekMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0, 0) // Create the thread's message queue.
		module, _, _ := getModuleHandle.Call(0)
		hook, _, err := setWindowsHook.Call(13, callback, module, 0)
		if hook == 0 {
			ready <- winError("install global keyboard listener", err)
			return
		}
		defer unhookWindowsHook.Call(hook)
		ready <- nil
		for {
			result, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
			if int32(result) <= 0 {
				return
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&message)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
		}
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return listener, nil
}

func (listener *keyboardListener) stop() {
	postThreadMessage.Call(listener.thread, 0x12, 0, 0)
	<-listener.done
}
