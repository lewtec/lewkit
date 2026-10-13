package main

import (
	"context"
	"math"

	"github.com/lewtec/lewkit/x/ui/world"
)

// tint is the color of the fully shown box. recolor walks it around the hue wheel.
type tint struct {
	R, G, B uint8
}

const hueTurns = 0.12

func spawnFill(_ context.Context, w *world.World) {
	world.Spawn(w, tint{R: 255})
}

func recolor(_ context.Context, w *world.World) {
	clock, ok := world.Read[world.Time](w)
	if !ok {
		return
	}
	red, green, blue := wash(clock.Elapsed * hueTurns)
	world.Query[tint](w).Each(func(_ world.Entity, color *tint) {
		color.R, color.G, color.B = red, green, blue
	})
}

// wash is one turn of the hue wheel. 0 is red.
func wash(turns float64) (uint8, uint8, uint8) {
	turns = math.Mod(turns, 1)
	if turns < 0 {
		turns += 1
	}
	sector := turns * 6
	index := int(sector)
	frac := sector - float64(index)
	var red, green, blue float64
	switch index {
	case 0:
		red, green, blue = 1, frac, 0
	case 1:
		red, green, blue = 1-frac, 1, 0
	case 2:
		red, green, blue = 0, 1, frac
	case 3:
		red, green, blue = 0, 1-frac, 1
	case 4:
		red, green, blue = frac, 0, 1
	default:
		red, green, blue = 1, 0, 1-frac
	}
	return byteOf(red), byteOf(green), byteOf(blue)
}

func byteOf(channel float64) uint8 {
	return uint8(channel*255 + 0.5)
}
