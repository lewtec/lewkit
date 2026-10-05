package main

import (
	"context"
	"math"

	"github.com/lewtec/lewkit/x/ui/world"
)

// side is the tutorial's PieceColor. White moves first, toward increasing x.
type side uint8

const (
	sideWhite side = iota
	sideBlack
)

func (s side) other() side {
	if s == sideWhite {
		return sideBlack
	}
	return sideWhite
}

func (s side) String() string {
	if s == sideWhite {
		return "White"
	}
	return "Black"
}

// kind is the tutorial's PieceType, in the same order.
type kind uint8

const (
	kindKing kind = iota
	kindQueen
	kindBishop
	kindKnight
	kindRook
	kindPawn
)

// piece is one piece on the board. x is the rank from White's back rank.
// y is the file. This is the tutorial's coordinate, not algebraic notation.
type piece struct {
	color side
	kind  kind
	x, y  uint8
}

// valid reports whether moving to (x, y) is legal. pieces includes the mover.
// The tutorial allows no castling, en passant, or promotion.
func (p piece) valid(x, y uint8, pieces []piece) bool {
	if c, ok := colorOf(x, y, pieces); ok && c == p.color {
		return false
	}
	switch p.kind {
	case kindKing:
		return absInt(int(p.x)-int(x)) <= 1 && absInt(int(p.y)-int(y)) <= 1 && (p.x != x || p.y != y)
	case kindQueen:
		return pathEmpty(p.x, p.y, x, y, pieces) && (absInt(int(p.x)-int(x)) == absInt(int(p.y)-int(y)) || p.x == x || p.y == y) && (p.x != x || p.y != y)
	case kindBishop:
		return pathEmpty(p.x, p.y, x, y, pieces) && absInt(int(p.x)-int(x)) == absInt(int(p.y)-int(y)) && (p.x != x || p.y != y)
	case kindKnight:
		dx, dy := absInt(int(p.x)-int(x)), absInt(int(p.y)-int(y))
		return (dx == 2 && dy == 1) || (dx == 1 && dy == 2)
	case kindRook:
		return pathEmpty(p.x, p.y, x, y, pieces) && (p.x == x || p.y == y) && (p.x != x || p.y != y)
	default:
		return p.pawn(x, y, pieces)
	}
}

func (p piece) pawn(x, y uint8, pieces []piece) bool {
	forward := 1
	start := uint8(1)
	enemy := sideBlack
	if p.color == sideBlack {
		forward = -1
		start = 6
		enemy = sideWhite
	}
	dx := int(x) - int(p.x)
	if dx == forward && p.y == y {
		if _, ok := colorOf(x, y, pieces); !ok {
			return true
		}
	}
	if p.x == start && dx == 2*forward && p.y == y && pathEmpty(p.x, p.y, x, y, pieces) {
		if _, ok := colorOf(x, y, pieces); !ok {
			return true
		}
	}
	if dx == forward && absInt(int(p.y)-int(y)) == 1 {
		if c, ok := colorOf(x, y, pieces); ok && c == enemy {
			return true
		}
	}
	return false
}

func pathEmpty(x0, y0, x1, y1 uint8, pieces []piece) bool {
	if x0 == x1 {
		for _, p := range pieces {
			if p.x == x0 && ((p.y > y0 && p.y < y1) || (p.y > y1 && p.y < y0)) {
				return false
			}
		}
	}
	if y0 == y1 {
		for _, p := range pieces {
			if p.y == y0 && ((p.x > x0 && p.x < x1) || (p.x > x1 && p.x < x0)) {
				return false
			}
		}
	}
	xDiff := absInt(int(x0) - int(x1))
	yDiff := absInt(int(y0) - int(y1))
	if xDiff == yDiff {
		for i := 1; i < xDiff; i++ {
			var x, y uint8
			switch {
			case x0 < x1 && y0 < y1:
				x, y = x0+uint8(i), y0+uint8(i)
			case x0 < x1 && y0 > y1:
				x, y = x0+uint8(i), y0-uint8(i)
			case x0 > x1 && y0 < y1:
				x, y = x0-uint8(i), y0+uint8(i)
			default:
				x, y = x0-uint8(i), y0-uint8(i)
			}
			if _, ok := colorOf(x, y, pieces); ok {
				return false
			}
		}
	}
	return true
}

func colorOf(x, y uint8, pieces []piece) (side, bool) {
	for _, p := range pieces {
		if p.x == x && p.y == y {
			return p.color, true
		}
	}
	return 0, false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// pose is where the mesh sits. piece.x and piece.y jump on a legal move.
// slidePieces walks pose toward that square at one unit per second, which
// is the tutorial's move_pieces rate, and snaps inside a tenth of a square.
type pose struct {
	x, z float32
}

func piecesPlugin(s *world.Sim) {
	s.System(world.Startup, createPieces)
	s.System(world.Update, slidePieces).After(colorSquares)
}

func slidePieces(_ context.Context, w *world.World) {
	clock, _ := world.Read[world.Time](w)
	step := float32(clock.Delta)
	if step < 0 {
		step = 0
	}
	world.Query[pose](w).Each(func(e world.Entity, at *pose) {
		body, ok := world.Query[piece](w).Get(e)
		if !ok {
			return
		}
		dx := float32(body.x) - at.x
		dz := float32(body.y) - at.z
		dist := float32(math.Hypot(float64(dx), float64(dz)))
		if dist < 0.1 || step == 0 {
			if dist < 0.1 {
				at.x, at.z = float32(body.x), float32(body.y)
			}
			return
		}
		move := step
		if move > dist {
			move = dist
		}
		at.x += dx / dist * move
		at.z += dz / dist * move
	})
}

func createPieces(_ context.Context, w *world.World) {
	back := []kind{kindRook, kindKnight, kindBishop, kindQueen, kindKing, kindBishop, kindKnight, kindRook}
	for y, k := range back {
		spawnPiece(w, sideWhite, k, 0, uint8(y))
		spawnPiece(w, sideBlack, k, 7, uint8(y))
	}
	for y := range uint8(8) {
		spawnPiece(w, sideWhite, kindPawn, 1, y)
		spawnPiece(w, sideBlack, kindPawn, 6, y)
	}
}

func spawnPiece(w *world.World, color side, k kind, x, y uint8) {
	world.Spawn2(w, piece{color: color, kind: k, x: x, y: y}, pose{x: float32(x), z: float32(y)})
}

func pieceList(w *world.World) []piece {
	return world.Query[piece](w).All()
}

func pieceAt(w *world.World, x, y uint8) (world.Entity, piece, bool) {
	return world.Query[piece](w).First(func(p piece) bool {
		return p.x == x && p.y == y
	})
}
