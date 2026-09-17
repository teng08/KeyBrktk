package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"
)

func TestDesktopAppearanceMatches(t *testing.T) {
	data, err := os.ReadFile("../KeybedMac/Sources/KeybedMac/SoundPreset.swift")
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`detail: "([^"]+)", symbol: "[^"]+", color: \.init\(srgbRed: ([0-9.]+), green: ([0-9.]+), blue: ([0-9.]+)`).FindAllSubmatch(data, -1)
	if len(matches) != len(presetLooks) || len(presetLooks) != len(presetNames) {
		t.Fatal("desktop visual catalogs differ")
	}
	for index, match := range matches {
		look := presetLooks[index]
		if string(match[1]) != look.detail {
			t.Fatalf("description differs for %s", presetNames[index])
		}
		for channel, want := range []float64{look.r, look.g, look.b} {
			got, err := strconv.ParseFloat(string(match[channel+2]), 64)
			if err != nil || got != want {
				t.Fatalf("accent differs for %s", presetNames[index])
			}
		}
	}
}

func TestSoundCardsFitViewport(t *testing.T) {
	for _, width := range []int{580, 650, 687, 704, 1100} {
		cards := soundCardLayout(width)
		for index, r := range cards {
			if r.x < 0 || r.y < 0 || r.x+r.w > width || r.y+r.h > 342 || r.w <= 0 || r.h != 62 {
				t.Fatalf("%s lies outside library width %d", presetNames[index], width)
			}
			for other, q := range cards {
				if index != other && r.x < q.x+q.w && q.x < r.x+r.w && r.y < q.y+q.h && q.y < r.y+r.h {
					t.Fatal("sound cards overlap")
				}
			}
		}
		if cards[9] != (layoutRect{0, 0, width, 62}) {
			t.Fatal("featured sound differs from Mac")
		}
		for index := 10; index <= 12; index++ {
			if cards[index].y != 70 {
				t.Fatal("new switches should be immediately visible")
			}
		}
	}
}

func TestFloatingCounterFitsDisplays(t *testing.T) {
	for _, work := range []layoutRect{{0, 0, 1920, 1040}, {-1920, -180, 1920, 1080}, {0, 0, 1366, 728}, {0, 0, 320, 220}} {
		for _, height := range []int{36, 66} {
			for _, point := range [][2]int{{work.x, work.y}, {work.x + work.w - 1, work.y + work.h - 1}, {work.x + work.w/2, work.y + work.h/2}} {
				x, y := floatingOrigin(point[0], point[1], work, 218, height)
				if x < work.x+8 || y < work.y+8 || x+218 > work.x+work.w-8 || y+height > work.y+work.h-8 {
					t.Fatal("floating counter leaves display work area")
				}
			}
		}
	}
}

func TestCounterResetClearsPulse(t *testing.T) {
	var counter typingCounter
	now := time.Now()
	counter.record(now)
	if count, last := counter.activity(now); count != 1 || last != now {
		t.Fatal("HUD activity did not track typing")
	}
	counter.reset()
	if count, last := counter.activity(now); count != 0 || !last.IsZero() {
		t.Fatal("reset did not clear count and HUD pulse")
	}
}
