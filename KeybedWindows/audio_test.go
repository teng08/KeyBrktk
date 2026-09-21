package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"sync"
	"testing"
	"time"
)

func testBank() *soundBank {
	bank := new(soundBank)
	for p := range bank {
		for mode := range bank[p] {
			for key := range bank[p][mode] {
				frames := make([]float32, 128)
				for i := range frames {
					frames[i] = float32(10+p*3+mode+key) / 100
				}
				bank[p][mode][key] = frames
			}
		}
	}
	return bank
}

func bankBytes(bank *soundBank) []byte {
	var data bytes.Buffer
	data.WriteString("KBPCM001")
	for _, value := range []uint32{sampleRate, uint32(len(presetNames)), uint32(len(intensityNames)), 12} {
		binary.Write(&data, binary.LittleEndian, value)
	}
	for p := range bank {
		for mode := range bank[p] {
			for _, frames := range bank[p][mode] {
				binary.Write(&data, binary.LittleEndian, uint32(len(frames)))
				binary.Write(&data, binary.LittleEndian, frames)
			}
		}
	}
	return data.Bytes()
}

func TestSoundBankFormat(t *testing.T) {
	data := bankBytes(testBank())
	bank, err := readBank(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if bank[len(presetNames)-1][2][11][0] != testBank()[len(presetNames)-1][2][11][0] {
		t.Fatal("roundtrip changed PCM")
	}
	for _, end := range []int{0, 7, 23, 25, len(data) - 1} {
		if _, err := readBank(bytes.NewReader(data[:end])); err == nil {
			t.Fatalf("accepted truncation at %d", end)
		}
	}
	for _, offset := range []int{0, 8, 12, 16, 20} {
		bad := append([]byte(nil), data...)
		bad[offset]++
		if _, err := readBank(bytes.NewReader(bad)); err == nil {
			t.Fatalf("accepted invalid header at %d", offset)
		}
	}
	bad := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(bad[24:28], 0xffffffff)
	if _, err := readBank(bytes.NewReader(bad)); err == nil {
		t.Fatal("accepted oversized sample")
	}
	for _, value := range []float32{float32(math.NaN()), float32(math.Inf(1)), 1.1} {
		bad = append([]byte(nil), data...)
		binary.LittleEndian.PutUint32(bad[28:32], math.Float32bits(value))
		if _, err := readBank(bytes.NewReader(bad)); err == nil {
			t.Fatal("accepted invalid PCM")
		}
	}
	if _, err := readBank(bytes.NewReader(append(data, 0))); err == nil {
		t.Fatal("accepted trailing data")
	}
	legacy := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(legacy[12:16], 10)
	if _, err := readBank(bytes.NewReader(legacy)); err == nil {
		t.Fatal("accepted a bank from the older 10-preset release")
	}
}

func TestProductionBank(t *testing.T) {
	path := os.Getenv("KEYBED_SOUND_BANK")
	if path == "" {
		t.Skip("set KEYBED_SOUND_BANK to validate the exported production samples")
	}
	bank, err := loadBank(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyBank(bank); err != nil {
		t.Fatal(err)
	}
}

func TestMixerSettingsAndKeyMapping(t *testing.T) {
	bank := testBank()
	for p := range presetNames {
		for mode := range intensityNames {
			for kind := 0; kind < 4; kind++ {
				for _, release := range []bool{false, true} {
					m := newMixer(bank)
					m.configure(settings{Preset: p, Intensity: mode, Volume: 100, Releases: true})
					m.enqueue(keyEvent{kind: kind, release: release})
					output := make([]int16, 256)
					m.render(output)
					key := 1
					if kind != 0 {
						key = 4 + kind
					}
					if release {
						key = 8 + kind
					}
					want := int16(bank[p][mode][key][0] * 32767)
					if output[0] != want || output[128] != 0 {
						t.Fatalf("mapping %d/%d/%d/%v: %d vs %d", p, mode, kind, release, output[0], want)
					}
				}
			}
		}
	}
	for _, config := range []settings{{Muted: true, Volume: 100, Releases: true}, {Volume: 0, Releases: true}, {Volume: 100, Releases: false}} {
		m := newMixer(bank)
		m.configure(config)
		m.enqueue(keyEvent{release: true})
		output := make([]int16, 128)
		m.render(output)
		for _, value := range output {
			if value != 0 {
				t.Fatal("mute, zero volume or disabled releases were audible")
			}
		}
	}
}

func TestKeyStateIgnoresHeldKeyRepeats(t *testing.T) {
	var state keyState
	if !state.press(65) || state.press(65) {
		t.Fatal("held key was not limited to its first key-down")
	}
	if !state.release(65) || state.release(65) {
		t.Fatal("key release did not end exactly one press cycle")
	}
	if !state.press(65) {
		t.Fatal("key did not re-arm after release")
	}
	if state.press(256) || state.release(256) {
		t.Fatal("out-of-range key changed state")
	}
}

func TestOverlappingVoicesAndAllocationFreeRender(t *testing.T) {
	m := newMixer(testBank())
	m.configure(settings{Volume: 100})
	for i := 0; i < 32; i++ {
		m.enqueue(keyEvent{})
	}
	output := make([]int16, 256)
	m.render(output)
	if output[0] != 32767 || output[128] != 0 {
		t.Fatal("voices did not overlap, clip safely or complete")
	}
	if allocations := testing.AllocsPerRun(100, func() { m.enqueue(keyEvent{}); m.render(output) }); allocations != 0 {
		t.Fatalf("render allocated %v times", allocations)
	}
	for i := 0; i < 10000; i++ {
		m.enqueue(keyEvent{})
	} // A full queue never blocks input.
	m.render(output)
}

func TestConcurrentSettingsAndEvents(t *testing.T) {
	m := newMixer(testBank())
	var wait sync.WaitGroup
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for j := 0; j < 1000; j++ {
				m.configure(settings{Preset: j % len(presetNames), Intensity: j % len(intensityNames), Volume: 72})
				m.enqueue(keyEvent{})
			}
		}()
	}
	output := make([]int16, 256)
	for i := 0; i < 1000; i++ {
		m.render(output)
	}
	wait.Wait()
}

func TestTypingCounter(t *testing.T) {
	var counter typingCounter
	now := time.Unix(100, 0)
	counter.record(now)
	counter.record(now.Add(2999 * time.Millisecond))
	if counter.snapshot(now.Add(2999*time.Millisecond)) != 2 {
		t.Fatal("reset too early")
	}
	if counter.snapshot(now.Add(3*time.Second)) != 0 {
		t.Fatal("did not reset at three seconds")
	}
	counter.record(now.Add(3 * time.Second))
	if counter.snapshot(now.Add(3*time.Second)) != 1 {
		t.Fatal("new window did not restart at one")
	}
	counter.record(now.Add(100 * time.Second))
	if counter.snapshot(now.Add(100*time.Second)) != 1 {
		t.Fatal("hidden/idle counter did not expire")
	}
	var concurrent typingCounter
	var wait sync.WaitGroup
	for i := 0; i < 4; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for j := 0; j < 1000; j++ {
				concurrent.record(now)
			}
		}()
	}
	wait.Wait()
	if concurrent.snapshot(now) != 4000 {
		t.Fatal("concurrent events lost")
	}
}
