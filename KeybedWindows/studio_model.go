package main

// The desktop apps share this visual catalog as well as their audio catalog.
// Keep the original preset indices stable for saved settings and sound banks.
type presetAppearance struct {
	detail  string
	r, g, b float64
}

var presetLooks = [...]presetAppearance{
	{"The original · crisp & clean", 1, .60, .35},
	{"A warm, satisfying bump", .82, .66, .49},
	{"Bright mechanical clicks", .43, .72, 1},
	{"Low, rounded & punchy", .68, .57, 1},
	{"Soft taps with a glassy body", .89, .82, .65},
	{"Metallic, old-school clacks", .77, .82, .85},
	{"Tiny, playful water drops", .36, .86, .77},
	{"A little arcade in every key", 1, .48, .65},
	{"Quick, cheerful little chirps", .75, .90, .41},
	{"Robot vocals · skibiddy, toilet, dop dop", .46, .95, .65},
	{"Recorded tactile pops & deep body", 1, .76, .43},
	{"Recorded warm, rounded clacks", .96, .88, .69},
	{"Recorded smooth, crisp taps", .27, .87, .88},
}

type layoutRect struct{ x, y, w, h int }

// Match the Mac studio: featured vocals, new recorded switches, then originals.
func soundCardLayout(width int) [len(presetNames)]layoutRect {
	var cards [len(presetNames)]layoutRect
	cards[9] = layoutRect{0, 0, width, 62}
	order := [...]int{10, 11, 12, 0, 1, 2, 3, 4, 5, 6, 7, 8}
	for position, index := range order {
		column := position % 3
		x := column * (width + 10) / 3
		right := (column+1)*(width+10)/3 - 10
		if column == 2 {
			right = width
		}
		cards[index] = layoutRect{x, 70 + position/3*70, right - x, 62}
	}
	return cards
}

// Windows coordinates grow downwards. Clamp against the cursor's monitor work
// area, including negative coordinates on displays left/above the main one.
func floatingOrigin(cursorX, cursorY int, work layoutRect, width, height int) (int, int) {
	x, y := cursorX+18, cursorY+18
	if x+width > work.x+work.w-8 {
		x = cursorX - width - 18
	}
	if y+height > work.y+work.h-8 {
		y = cursorY - height - 18
	}
	x = clamp(x, work.x+8, max(work.x+8, work.x+work.w-width-8))
	y = clamp(y, work.y+8, max(work.y+8, work.y+work.h-height-8))
	return x, y
}

func clamp(value, low, high int) int { return max(low, min(value, high)) }
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
