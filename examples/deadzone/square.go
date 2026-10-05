package main

import (
	"context"
	"math"

	"github.com/lewtec/lewkit/x/ui/world"
)

// spot is where the square sits in the fully shown box.
// 0 is the start of the travel, 1 is the far edge.
type spot struct {
	X, Y float32
}

// glide is the travel per second, in the same 0..1 span as spot.
type glide struct {
	X, Y float32
}

// tint is the square's color. recolor walks it around the hue wheel.
type tint struct {
	R, G, B uint8
}

const (
	paceX    = float32(0.31)
	paceY    = float32(0.23)
	hueTurns = 0.12
	// squareShare is the square's side as a fraction of the shorter safe edge.
	squareShare = float32(0.22)
)

func spawnSquare(_ context.Context, w *world.World) {
	entity := world.Spawn2(w, spot{}, glide{X: paceX, Y: paceY})
	world.Insert(w, entity, tint{R: 255})
}

func drift(_ context.Context, w *world.World) {
	clock, ok := world.Read[world.Time](w)
	if !ok {
		return
	}
	dt := float32(clock.Delta)
	motion := world.Query[glide](w)
	world.Query[spot](w).Each(func(entity world.Entity, at *spot) {
		step := motion.Mut(entity)
		if step == nil {
			return
		}
		at.X = bounce(at.X, &step.X, dt)
		at.Y = bounce(at.Y, &step.Y, dt)
	})
}

func bounce(at float32, pace *float32, dt float32) float32 {
	at += *pace * dt
	if at < 0 {
		at = 0
		*pace = -*pace
	} else if at > 1 {
		at = 1
		*pace = -*pace
	}
	return at
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
