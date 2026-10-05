package main

import (
	"context"

	"github.com/lewtec/lewkit/x/ui/world"
)

// square is one board cell. x and y match [piece].
type square struct {
	x, y uint8
}

func (s square) light() bool { return (int(s.x)+int(s.y)+1)%2 == 0 }

// shade is the square tint colorSquares writes for the view.
type shade uint8

const (
	shadeDark shade = iota
	shadeLight
	shadeHover
	shadeSelected
)

// click is the pointer sample for this frame.
// The host replaces it before Frame. selectSquare clears it.
type click struct {
	x, y uint8
	hit  bool
	down bool
}

// cursor is the square under the pointer. colorSquares reads it.
type cursor struct {
	x, y uint8
	on   bool
}

type selectedSquare struct {
	e  world.Entity
	ok bool
}

type selectedPiece struct {
	e  world.Entity
	ok bool
}

type playerTurn struct{ side side }

func (t *playerTurn) change() {
	if t == nil {
		return
	}
	t.side = t.side.other()
}

// taken marks a piece movePiece captured. despawnTaken removes it.
type taken struct{}

// picked is a click selectSquare accepted. movePiece and selectPiece
// each keep a cursor, so both observe the one send.
type picked struct{}

// resetSelected asks the later system to clear the selection.
// movePiece sends it. The reader that has not caught up can still see
// it on the next frame.
type resetSelected struct{}

// outcome is set when a king is taken. The other side won.
type outcome struct {
	winner side
	over   bool
}

// banner is the line the tutorial drew as UI text.
type banner struct{ text string }

func boardPlugin(s *world.Sim) {
	world.Init(s.World, playerTurn{side: sideWhite})
	world.Init(s.World, cpuSide{side: sideBlack, play: true})
	world.Init(s.World, selectedSquare{})
	world.Init(s.World, selectedPiece{})
	world.Init(s.World, click{})
	world.Init(s.World, cursor{})
	s.System(world.Startup, createBoard)
	// The tutorial's order: select the square, try the move, then select
	// a piece, then reset. Despawn runs after the CPU plugin, which
	// inserts itself before despawnTaken. Color runs after the captures leave.
	s.Chain(world.Update,
		selectSquare,
		movePiece,
		selectPiece,
		resetSelectedSystem,
		despawnTaken,
		colorSquares,
	)
	s.System(world.PostUpdate, announceTurn)
}

func createBoard(_ context.Context, w *world.World) {
	for x := range uint8(8) {
		for y := range uint8(8) {
			world.Spawn(w, square{x: x, y: y})
		}
	}
}

func selectSquare(_ context.Context, w *world.World) {
	if over, ok := world.Read[outcome](w); ok && over.over {
		if in, ok := world.Read[click](w); ok && in.down {
			world.Put(w, click{})
		}
		return
	}
	in, ok := world.Read[click](w)
	if !ok || !in.down {
		return
	}
	world.Put(w, click{})
	sel := world.Mut[selectedSquare](w)
	if sel == nil {
		return
	}
	if !in.hit {
		sel.ok = false
		if piece := world.Mut[selectedPiece](w); piece != nil {
			piece.ok = false
		}
		return
	}
	e, found := squareAt(w, in.x, in.y)
	if !found {
		sel.ok = false
		return
	}
	sel.e = e
	sel.ok = true
	world.Send(w, picked{})
}

func squareAt(w *world.World, x, y uint8) (world.Entity, bool) {
	found, _, ok := world.Query[square](w).First(func(sq square) bool {
		return sq.x == x && sq.y == y
	})
	return found, ok
}

func movePiece(_ context.Context, w *world.World) {
	if len(world.Messages[picked](w)) == 0 {
		return
	}
	pieceSel, _ := world.Read[selectedPiece](w)
	if !pieceSel.ok {
		return
	}
	sqSel, _ := world.Read[selectedSquare](w)
	if !sqSel.ok {
		return
	}
	sq, ok := world.Query[square](w).Get(sqSel.e)
	if !ok {
		return
	}
	all := pieceList(w)
	mover := world.Query[piece](w).Mut(pieceSel.e)
	if mover == nil {
		return
	}
	if mover.valid(sq.x, sq.y, all) {
		applyMove(w, mover, sq.x, sq.y)
	}
	world.Send(w, resetSelected{})
}

// applyMove captures an opposite piece on (x, y), moves mover, and
// hands the turn over. Taking a king ends the game before a later
// system in this frame can move the captured side.
func applyMove(w *world.World, mover *piece, x, y uint8) {
	if mover == nil {
		return
	}
	capturedKing := false
	var victim side
	world.Query[piece](w).Read(func(e world.Entity, other piece) {
		if other.x != x || other.y != y || other.color == mover.color {
			return
		}
		world.Insert(w, e, taken{})
		if other.kind == kindKing {
			capturedKing = true
			victim = other.color
		}
	})
	mover.x, mover.y = x, y
	if turn := world.Mut[playerTurn](w); turn != nil {
		turn.change()
	}
	if capturedKing {
		world.Put(w, outcome{winner: victim.other(), over: true})
	}
}

func selectPiece(_ context.Context, w *world.World) {
	if len(world.Messages[picked](w)) == 0 {
		return
	}
	sqSel, _ := world.Read[selectedSquare](w)
	if !sqSel.ok {
		return
	}
	holding := world.Mut[selectedPiece](w)
	if holding == nil || holding.ok {
		return
	}
	sq, ok := world.Query[square](w).Get(sqSel.e)
	if !ok {
		return
	}
	turn, _ := world.Read[playerTurn](w)
	world.Query[piece](w).Read(func(e world.Entity, p piece) {
		if holding.ok || p.x != sq.x || p.y != sq.y || p.color != turn.side {
			return
		}
		holding.e = e
		holding.ok = true
	})
}

func resetSelectedSystem(_ context.Context, w *world.World) {
	if len(world.Messages[resetSelected](w)) == 0 {
		return
	}
	if sel := world.Mut[selectedSquare](w); sel != nil {
		*sel = selectedSquare{}
	}
	if sel := world.Mut[selectedPiece](w); sel != nil {
		*sel = selectedPiece{}
	}
}

func despawnTaken(_ context.Context, w *world.World) {
	var doomed []world.Entity
	world.With[piece, taken](world.Query[piece](w)).Read(func(e world.Entity, p piece) {
		if p.kind == kindKing {
			world.Put(w, outcome{winner: p.color.other(), over: true})
		}
		doomed = append(doomed, e)
	})
	for _, e := range doomed {
		w.Despawn(e)
	}
}

func colorSquares(_ context.Context, w *world.World) {
	cur, _ := world.Read[cursor](w)
	sel, _ := world.Read[selectedSquare](w)
	world.Query[square](w).Read(func(e world.Entity, sq square) {
		next := shadeDark
		if sq.light() {
			next = shadeLight
		}
		if sel.ok && sel.e == e {
			next = shadeSelected
		}
		if cur.on && cur.x == sq.x && cur.y == sq.y {
			next = shadeHover
		}
		world.Insert(w, e, next)
	})
}

func announceTurn(_ context.Context, w *world.World) {
	if end, ok := world.Read[outcome](w); ok && end.over {
		world.Put(w, banner{text: end.winner.String() + " won. Thanks for playing!"})
		return
	}
	if !world.Written[playerTurn](w) {
		if _, ok := world.Read[banner](w); ok {
			return
		}
	}
	turn, ok := world.Read[playerTurn](w)
	if !ok {
		return
	}
	world.Put(w, banner{text: "Next move: " + turn.side.String()})
}
