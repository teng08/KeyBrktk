//go:build windows

package main

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"
)

type waveFormat struct {
	tag, channels          uint16
	rate, bytesPerSecond   uint32
	alignment, bits, extra uint16
}

type waveHeader struct {
	data             uintptr
	length, recorded uint32
	user             uintptr
	flags, loops     uint32
	next, reserved   uintptr
}

type outputState struct {
	status atomic.Value
	done   chan struct{}
}

func startOutput(m *mixer, stop <-chan struct{}) *outputState {
	state := &outputState{done: make(chan struct{})}
	state.status.Store("Starting audio…")
	go func() {
		defer close(state.done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			err := streamOutput(m, stop, &state.status)
			if err == nil {
				return
			}
			state.status.Store(fmt.Sprintf("Audio unavailable (%v). Retrying…", err))
			select {
			case <-stop:
				return
			case <-time.After(time.Second):
			}
		}
	}()
	return state
}

func streamOutput(m *mixer, stop <-chan struct{}, status *atomic.Value) error {
	event, _, err := createEvent.Call(0, 0, 0, 0)
	if event == 0 {
		return winError("create audio event", err)
	}
	defer closeHandle.Call(event)
	format := waveFormat{tag: 1, channels: 1, rate: sampleRate, bytesPerSecond: sampleRate * 2, alignment: 2, bits: 16}
	var device uintptr
	result, _, _ := waveOutOpen.Call(uintptr(unsafe.Pointer(&device)), 0xffffffff, uintptr(unsafe.Pointer(&format)), event, 0, 0x50000)
	if result != 0 {
		return fmt.Errorf("open output device: code %d", result)
	}
	const framesPerBuffer = 256
	headerSize := unsafe.Sizeof(waveHeader{})
	var blocks [3]uintptr
	var prepared [3]bool
	defer func() {
		waveOutReset.Call(device)
		for i, block := range blocks {
			if block != 0 {
				if prepared[i] {
					result, _, _ := waveOutUnprepare.Call(device, block, headerSize)
					// A broken driver may still own the buffer. Never free it early.
					if result != 0 {
						continue
					}
				}
				virtualFree.Call(block, 0, 0x8000)
			}
		}
		waveOutClose.Call(device)
	}()
	for i := range blocks {
		block, _, err := virtualAlloc.Call(0, headerSize+framesPerBuffer*2, 0x3000, 0x04)
		if block == 0 {
			return winError("allocate audio buffer", err)
		}
		blocks[i] = block
		header := (*waveHeader)(unsafe.Pointer(block))
		header.data = block + headerSize
		header.length = framesPerBuffer * 2
		result, _, _ = waveOutPrepare.Call(device, block, headerSize)
		if result != 0 {
			return fmt.Errorf("prepare audio buffer: code %d", result)
		}
		prepared[i] = true
		m.render(unsafe.Slice((*int16)(unsafe.Pointer(header.data)), framesPerBuffer))
		result, _, _ = waveOutWrite.Call(device, block, headerSize)
		if result != 0 {
			return fmt.Errorf("queue audio buffer: code %d", result)
		}
	}
	status.Store("Listening globally · audio ready")
	lastCompletion := time.Now()
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		result, _, err = waitForSingleObject.Call(event, 100)
		if result == 0xffffffff {
			return winError("wait for audio device", err)
		}
		for _, block := range blocks {
			header := (*waveHeader)(unsafe.Pointer(block))
			if atomic.LoadUint32(&header.flags)&1 == 0 {
				continue
			}
			m.render(unsafe.Slice((*int16)(unsafe.Pointer(header.data)), framesPerBuffer))
			result, _, _ = waveOutWrite.Call(device, block, headerSize)
			if result != 0 {
				return fmt.Errorf("write audio buffer: code %d", result)
			}
			lastCompletion = time.Now()
		}
		if time.Since(lastCompletion) > 2*time.Second {
			return fmt.Errorf("output device stopped responding")
		}
	}
}
