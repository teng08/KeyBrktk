package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const sampleRate = 44100

// Append presets so saved selections from earlier releases retain their meaning.
var presetNames = [...]string{"Alpaca Linear", "Tactile Brown", "Clicky Blue", "Deep Thock", "Creamy Marble", "Typewriter", "Bubble Pop", "Pixel Tap", "Birdy Chirp", "Skibiddy Toilet", "Holy Pandas", "NovelKeys Creams", "Turquoise Tealios"}
var intensityNames = [...]string{"Balanced", "Aggressive", "Extreme"}

// Format: magic, sample rate, preset/mode/key counts, then length + float32 PCM
// for each preset/mode/key. The Mac exporter supplies the exact production bank.
type soundBank [len(presetNames)][len(intensityNames)][12][]float32

func readBank(reader io.Reader) (*soundBank, error) {
	data, err := io.ReadAll(io.LimitReader(reader, 16*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 16*1024*1024 {
		return nil, fmt.Errorf("sound bank exceeds size limit")
	}
	if len(data) < 24 || string(data[:8]) != "KBPCM001" {
		return nil, fmt.Errorf("invalid Keybed sound bank")
	}
	for i, want := range []uint32{sampleRate, uint32(len(presetNames)), uint32(len(intensityNames)), 12} {
		if binary.LittleEndian.Uint32(data[8+i*4:12+i*4]) != want {
			return nil, fmt.Errorf("sound bank does not match this release (expected %d presets); extract the executable and sound bank from the same ZIP", len(presetNames))
		}
	}
	bank := new(soundBank)
	position := 24
	for p := range bank {
		for mode := range bank[p] {
			for key := range bank[p][mode] {
				if position+4 > len(data) {
					return nil, io.ErrUnexpectedEOF
				}
				count := int(binary.LittleEndian.Uint32(data[position : position+4]))
				position += 4
				if count < 1 || count > sampleRate || count*4 > len(data)-position {
					return nil, fmt.Errorf("invalid or truncated sample %d/%d/%d", p, mode, key)
				}
				frames := make([]float32, count)
				for i := range frames {
					frames[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[position : position+4]))
					position += 4
					if math.IsNaN(float64(frames[i])) || math.IsInf(float64(frames[i]), 0) || math.Abs(float64(frames[i])) > 1 {
						return nil, fmt.Errorf("invalid PCM sample")
					}
				}
				bank[p][mode][key] = frames
			}
		}
	}
	if position != len(data) {
		return nil, fmt.Errorf("unexpected trailing sound bank data")
	}
	return bank, nil
}

func loadBank(path string) (*soundBank, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w; extract the entire download before opening Keybed.exe", path, err)
	}
	defer file.Close()
	return readBank(file)
}

type keyEvent struct {
	kind    int
	release bool
}
type voice struct {
	frames   []float32
	position int
}

type mixer struct {
	bank               *soundBank
	events             chan keyEvent
	voices             [32]voice
	nextVoice, variant int
	preset, intensity  atomic.Int32
	volume             atomic.Uint32
	muted, releases    atomic.Bool
}

func newMixer(bank *soundBank) *mixer {
	m := &mixer{bank: bank, events: make(chan keyEvent, 256)}
	m.configure(defaultSettings())
	return m
}

func (m *mixer) configure(settings settings) {
	settings.sanitize()
	m.preset.Store(int32(settings.Preset))
	m.intensity.Store(int32(settings.Intensity))
	m.volume.Store(math.Float32bits(float32(settings.Volume) / 100))
	m.muted.Store(settings.Muted)
	m.releases.Store(settings.Releases)
}

// The keyboard hook never decodes audio, writes files or waits for playback.
func (m *mixer) enqueue(event keyEvent) {
	select {
	case m.events <- event:
	default:
	}
}

func (m *mixer) add(event keyEvent) {
	if m.muted.Load() || (event.release && !m.releases.Load()) {
		return
	}
	key := 0
	if event.release {
		key = 8 + event.kind
	} else if event.kind != 0 {
		key = 4 + event.kind
	} else {
		m.variant = (m.variant + 1) % 5
		key = m.variant
	}
	m.voices[m.nextVoice] = voice{frames: m.bank[m.preset.Load()][m.intensity.Load()][key]}
	m.nextVoice = (m.nextVoice + 1) % len(m.voices)
}

// Only the audio thread owns the voice list. Render performs no allocations.
func (m *mixer) render(output []int16) {
	for i := 0; i < cap(m.events); i++ {
		select {
		case event := <-m.events:
			m.add(event)
		default:
			i = cap(m.events)
		}
	}
	gain := math.Float32frombits(m.volume.Load())
	if m.muted.Load() {
		gain = 0
	}
	for frame := range output {
		var value float32
		for i := range m.voices {
			v := &m.voices[i]
			if v.position < len(v.frames) {
				value += v.frames[v.position]
				v.position++
			}
		}
		value *= gain
		if value > 1 {
			value = 1
		} else if value < -1 {
			value = -1
		}
		output[frame] = int16(value * 32767)
	}
}

func verifyBank(bank *soundBank) error {
	output := make([]int16, sampleRate+512)
	signatures := make(map[string]bool)
	checked := 0
	for p := range bank {
		for mode := range bank[p] {
			for key, sample := range bank[p][mode] {
				m := newMixer(bank)
				m.voices[0] = voice{frames: sample}
				m.render(output)
				onset, last := -1, -1
				for i, frame := range output {
					if frame > 32 || frame < -32 {
						if onset == -1 {
							onset = i
						}
						last = i
					}
				}
				if onset < 0 || onset > 44 {
					return fmt.Errorf("sample attack check failed: %s/%s/%d", presetNames[p], intensityNames[mode], key)
				}
				if p == 9 && key == 6 && last <= sampleRate/2 {
					return fmt.Errorf("Skibiddy Toilet phrase is truncated")
				}
				if mode == 1 && key == 0 {
					var signature bytes.Buffer
					binary.Write(&signature, binary.LittleEndian, output[:512])
					signatures[signature.String()] = true
				}
				checked++
			}
		}
	}
	if len(signatures) != len(presetNames) {
		return fmt.Errorf("presets do not have distinct waveforms")
	}
	fmt.Printf("PASS: %d presets, %d preloaded sounds, distinct waveforms and attacks within 1 ms.\n", len(presetNames), checked)
	return nil
}

// No typed text is retained: only a count and timestamps for a three-second window.
type typingCounter struct {
	mu          sync.Mutex
	count       int
	start, last time.Time
}

func (c *typingCounter) record(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.start.IsZero() || now.Sub(c.start) >= 3*time.Second {
		c.count = 0
		c.start = now
	}
	c.count++
	c.last = now
}

func (c *typingCounter) snapshot(now time.Time) int {
	count, _ := c.activity(now)
	return count
}

func (c *typingCounter) activity(now time.Time) (int, time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.start.IsZero() && now.Sub(c.start) >= 3*time.Second {
		c.count = 0
		c.start = time.Time{}
	}
	return c.count, c.last
}

func (c *typingCounter) reset() {
	c.mu.Lock()
	c.count, c.start, c.last = 0, time.Time{}, time.Time{}
	c.mu.Unlock()
}
