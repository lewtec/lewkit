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

// ink is the logo color. A wall hit walks the palette. A corner turns it white.
type ink struct {
	R, G, B uint8
	N       uint8
}

const (
	paceX = float32(0.37)
	paceY = float32(0.18)
	// logoShare is the logo width as a fraction of the shorter usable edge.
	logoShare = float32(0.34)
)

// kick is the speed a wall assigns. The color index picks the entry,
// and the other axis picks three slots further along, so a bounce
// leaves with a new slope.
var kick = []float32{0.21, 0.29, 0.34, 0.16, 0.41, 0.25}

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
	entity := world.Spawn2(w, spot{}, glide{X: paceX, Y: paceY})
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
		} else {
			color.N = (color.N + 1) % uint8(len(dvdColors))
			next := dvdColors[color.N]
			color.R, color.G, color.B = next.R, next.G, next.B
		}
		if hitX {
			kickPace(&step.X, color.N)
		}
		if hitY {
			kickPace(&step.Y, color.N+3)
		}
	})
}

// bounce moves at by pace*dt and folds it back into 0..1.
// A fold flips pace. The bool is true when a wall was hit.
func bounce(at float32, pace *float32, dt float32) (float32, bool) {
	at += *pace * dt
	hit := false
	for range 8 {
		if at >= 0 && at <= 1 {
			return at, hit
		}
		if at < 0 {
			at = -at
		} else {
			at = 2 - at
		}
		*pace = -*pace
		hit = true
	}
	if at < 0 {
		at = 0
	} else if at > 1 {
		at = 1
	}
	return at, hit
}

func kickPace(pace *float32, n uint8) {
	sign := float32(1)
	if *pace < 0 {
		sign = -1
	}
	*pace = sign * kick[int(n)%len(kick)]
}
