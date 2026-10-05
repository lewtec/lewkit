package gui

import (
	"sort"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Detect returns the largest axis-aligned box inside window that misses
// every dead zone. Dead zones are navbars and notches in the same
// coordinates as window. The box is the part of the window that is fully
// shown. No zones returns window. A covered window returns the zero rect.
// Equal areas keep the box with the larger shorter side, then the wider
// one, then the taller one, then the topmost, then the leftmost.
func Detect(window Rect, dead []Rect) Rect {
	if window.Width <= 0 || window.Height <= 0 {
		return Rect{}
	}
	zones := coverZones(window, dead)
	if len(zones) == 0 {
		return window
	}
	xs := make([]float32, 0, 2+2*len(zones))
	xs = append(xs, window.X, window.X+window.Width)
	for _, zone := range zones {
		xs = append(xs, zone.X, zone.X+zone.Width)
	}
	xs = uniqueFloats(xs)
	limitY := band{window.Y, window.Y + window.Height}
	var best Rect
	var bestArea float64
	for leftIndex := 0; leftIndex < len(xs); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(xs); rightIndex++ {
			left, right := xs[leftIndex], xs[rightIndex]
			width := right - left
			if width <= 0 {
				continue
			}
			blocked := make([]band, 0, len(zones))
			for _, zone := range zones {
				if zone.X < right && zone.X+zone.Width > left {
					blocked = append(blocked, band{zone.Y, zone.Y + zone.Height})
				}
			}
			for _, open := range openBands(limitY, mergeBands(blocked)) {
				height := open.end - open.start
				if height <= 0 {
					continue
				}
				next := Rect{left, open.start, width, height}
				area := float64(width) * float64(height)
				if betterBox(next, area, best, bestArea) {
					best = next
					bestArea = area
				}
			}
		}
	}
	return best
}

// Hint is the fully shown box of its layout slot.
// Dead holds navbars and notches in local coordinates, origin at the top left.
// Layout gives [Child] the box from [Detect], so a child that honors
// constraints stays inside it. Paint clips to that box.
// The hint fills the incoming constraints. [Hint.Shown] is the box after Layout.
type Hint struct {
	Child Node
	Dead  []Rect

	box Rect
}

// Shown is the box [Detect] chose at the last Layout, in local coordinates.
func (hint *Hint) Shown() Rect {
	if hint == nil {
		return Rect{}
	}
	return hint.box
}

func (hint *Hint) Layout(constraints BoxConstraints) Size {
	if hint == nil {
		return Size{}
	}
	size := constraints.Constrain(Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	hint.box = Detect(Rect{Width: size.Width, Height: size.Height}, hint.Dead)
	if hint.Child != nil {
		hint.Child.Layout(Tight(hint.box.Width, hint.box.Height))
	}
	return size
}

func (hint *Hint) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if hint == nil || hint.Child == nil || hint.box.Width <= 0 || hint.box.Height <= 0 {
		return accumulatorOf(picture)
	}
	shown := Rect{origin.X + hint.box.X, origin.Y + hint.box.Y, hint.box.Width, hint.box.Height}
	next := clip.Intersect(shown)
	if next.Width <= 0 || next.Height <= 0 {
		return accumulatorOf(picture)
	}
	return hint.Child.Paint(origin.Add(Offset{hint.box.X, hint.box.Y}), next, picture)
}

type band struct {
	start, end float32
}

func coverZones(window Rect, dead []Rect) []Rect {
	if len(dead) == 0 {
		return nil
	}
	zones := make([]Rect, 0, len(dead))
	for _, zone := range dead {
		if zone.Width <= 0 || zone.Height <= 0 {
			continue
		}
		clipped := zone.Intersect(window)
		if clipped.Width > 0 && clipped.Height > 0 {
			zones = append(zones, clipped)
		}
	}
	return zones
}

func betterBox(next Rect, nextArea float64, best Rect, bestArea float64) bool {
	if nextArea != bestArea {
		return nextArea > bestArea
	}
	if short := min(next.Width, next.Height); short != min(best.Width, best.Height) {
		return short > min(best.Width, best.Height)
	}
	if next.Width != best.Width {
		return next.Width > best.Width
	}
	if next.Height != best.Height {
		return next.Height > best.Height
	}
	if next.Y != best.Y {
		return next.Y < best.Y
	}
	return next.X < best.X
}

func uniqueFloats(values []float32) []float32 {
	if len(values) == 0 {
		return nil
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	count := 1
	for index := 1; index < len(values); index++ {
		if values[index] != values[count-1] {
			values[count] = values[index]
			count++
		}
	}
	return values[:count]
}

func mergeBands(bands []band) []band {
	if len(bands) == 0 {
		return nil
	}
	sort.Slice(bands, func(i, j int) bool { return bands[i].start < bands[j].start })
	merged := []band{bands[0]}
	for _, item := range bands[1:] {
		last := &merged[len(merged)-1]
		if item.start <= last.end {
			if item.end > last.end {
				last.end = item.end
			}
			continue
		}
		merged = append(merged, item)
	}
	return merged
}

func openBands(limit band, blocked []band) []band {
	var open []band
	cursor := limit.start
	for _, item := range blocked {
		if item.end <= cursor {
			continue
		}
		if item.start >= limit.end {
			break
		}
		if item.start > cursor {
			open = append(open, band{cursor, min(item.start, limit.end)})
		}
		if item.end > cursor {
			cursor = item.end
		}
		if cursor >= limit.end {
			return open
		}
	}
	if cursor < limit.end {
		open = append(open, band{cursor, limit.end})
	}
	return open
}
