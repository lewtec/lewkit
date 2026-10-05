package main

import (
	"context"

	"github.com/lewtec/lewkit/x/ui/world"
)

// spot is the logo's place in the usable box. 0 is the start, 1 the far edge.
type spot struct {
	X, Y float32
}

// glide is the travel per second across that same span.
type glide struct {
	X, Y float32
}

// ink is the logo color. A corner hit turns it white.
type ink struct {
	R, G, B uint8
	N       uint8
}

const (
	pace = float32(0.22)
	// logoShare is the logo width as a fraction of the shorter usable edge.
	logoShare = float32(0.34)
)

// dvdColors is the cycle a wall hit walks. Index 0 is the color at rest.
var dvdColors = []ink{
	{R: 226, G: 38, B: 48},
	{R: 36, G: 96, B: 230},
	{R: 36, G: 186, B: 78},
	{R: 240, G: 196, B: 40},
	{R: 196, G: 40, B: 186},
	{R: 36, G: 196, B: 206},
}

func spawnLogo(_ context.Context, w *world.World) {
	entity := world.Spawn2(w, spot{}, glide{X: pace, Y: pace})
	world.Insert(w, entity, dvdColors[0])
}

func drift(_ context.Context, w *world.World) {
	clock, ok := world.Read[world.Time](w)
	if !ok {
		return
	}
	dt := float32(clock.Delta)
	motion := world.Query[glide](w)
	colors := world.Query[ink](w)
	world.Query[spot](w).Each(func(entity world.Entity, at *spot) {
		step := motion.Mut(entity)
		if step == nil {
			return
		}
		var hitX, hitY bool
		at.X, hitX = bounce(at.X, &step.X, dt)
		at.Y, hitY = bounce(at.Y, &step.Y, dt)
		color := colors.Mut(entity)
		if color == nil || (!hitX && !hitY) {
			return
		}
		if hitX && hitY {
			color.R, color.G, color.B = 255, 255, 255
			return
		}
		color.N = (color.N + 1) % uint8(len(dvdColors))
		next := dvdColors[color.N]
		color.R, color.G, color.B = next.R, next.G, next.B
	})
}

func bounce(at float32, pace *float32, dt float32) (float32, bool) {
	at += *pace * dt
	if at < 0 {
		at = 0
		*pace = -*pace
		return at, true
	}
	if at > 1 {
		at = 1
		*pace = -*pace
		return at, true
	}
	return at, false
}
