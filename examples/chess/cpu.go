package main

import (
	"context"
	"sort"

	"github.com/lewtec/lewkit/x/ui/world"
)

// cpuSide is the side the computer plays. play is false for a board
// where both sides are clicked by hand. It is a resource, not a message:
// it has to still be there on the frame after White moves.
type cpuSide struct {
	side side
	play bool
}

func cpuPlugin(s *world.Sim) {
	s.System(world.Update, cpuMove).After(resetSelectedSystem).Before(despawnTaken)
}

// cpuMove plays one legal move when it is the computer's turn.
// It runs after the human move has cleared the selection, and before
// despawn, so a capture on either side leaves in this frame. A taken
// king already ended the game, and a taken piece is not a legal mover.
func cpuMove(_ context.Context, w *world.World) {
	if end, ok := world.Read[outcome](w); ok && end.over {
		return
	}
	brain, ok := world.Read[cpuSide](w)
	if !ok || !brain.play {
		return
	}
	turn, ok := world.Read[playerTurn](w)
	if !ok || turn.side != brain.side {
		return
	}
	if hold, _ := world.Read[selectedPiece](w); hold.ok {
		return
	}
	e, x, y, ok := chooseCPU(w, brain.side)
	if !ok {
		return
	}
	mover := world.Query[piece](w).Mut(e)
	if mover == nil {
		return
	}
	applyMove(w, mover, x, y)
}

type occupant struct {
	e world.Entity
	p piece
}

// living is every piece that has not been captured earlier in this frame.
func living(w *world.World) []occupant {
	var all []occupant
	world.Without[piece, taken](world.Query[piece](w)).Read(func(e world.Entity, p piece) {
		all = append(all, occupant{e: e, p: p})
	})
	return all
}

// chooseCPU returns one legal move. A capture outranks a quiet move, and
// a king outranks every other capture. Equal scores keep the earlier
// piece in rank-file order and the earlier destination, so a reply is
// the same on every run.
func chooseCPU(w *world.World, color side) (world.Entity, uint8, uint8, bool) {
	all := living(w)
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i].p, all[j].p
		if a.x != b.x {
			return a.x < b.x
		}
		if a.y != b.y {
			return a.y < b.y
		}
		return a.kind < b.kind
	})
	flat := make([]piece, len(all))
	for i := range all {
		flat[i] = all[i].p
	}
	var (
		bestE        world.Entity
		bestX, bestY uint8
		bestScore    int
		found        bool
	)
	for _, occ := range all {
		if occ.p.color != color {
			continue
		}
		for x := range uint8(8) {
			for y := range uint8(8) {
				if !occ.p.valid(x, y, flat) {
					continue
				}
				score := moveScore(occ.p, x, y, flat)
				if found && score <= bestScore {
					continue
				}
				found = true
				bestScore = score
				bestE = occ.e
				bestX, bestY = x, y
			}
		}
	}
	return bestE, bestX, bestY, found
}

func moveScore(mover piece, x, y uint8, pieces []piece) int {
	score := centerPull(x, y)*3 + advanceBonus(mover, x)
	for _, p := range pieces {
		if p.x == x && p.y == y && p.color != mover.color {
			score += captureValue(p.kind) * 100
			break
		}
	}
	return score
}

func captureValue(k kind) int {
	switch k {
	case kindQueen:
		return 9
	case kindRook:
		return 5
	case kindBishop, kindKnight:
		return 3
	case kindKing:
		return 40
	default:
		return 1
	}
}

// centerPull is 6 on the four center squares and 0 in a corner.
func centerPull(x, y uint8) int {
	return 6 - axisCenter(x) - axisCenter(y)
}

func axisCenter(v uint8) int {
	a := absInt(int(v) - 3)
	b := absInt(int(v) - 4)
	if b < a {
		return b
	}
	return a
}

func advanceBonus(mover piece, x uint8) int {
	switch mover.kind {
	case kindPawn:
		if mover.color == sideWhite {
			return int(x) - int(mover.x)
		}
		return int(mover.x) - int(x)
	case kindKnight, kindBishop:
		back := uint8(0)
		if mover.color == sideBlack {
			back = 7
		}
		if mover.x == back && x != back {
			return 2
		}
	}
	return 0
}
