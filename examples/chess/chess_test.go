package main

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/lewtec/lewkit/x/ui/world"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bootSim(t *testing.T) *world.Sim {
	t.Helper()
	sim := world.New()
	sim.Plugin(world.Group(boardPlugin, piecesPlugin, cpuPlugin))
	sim.Frame(t.Context())
	return sim
}

func clickSquare(t *testing.T, sim *world.Sim, x, y uint8) {
	t.Helper()
	world.Put(sim.World, cursor{x: x, y: y, on: true})
	world.Put(sim.World, click{x: x, y: y, hit: true, down: true})
	sim.Frame(t.Context())
}

// disableCPU leaves both colors to the clicks. The window keeps Black on.
func disableCPU(sim *world.Sim) {
	world.Put(sim.World, cpuSide{})
}

func TestStartupBoard(t *testing.T) {
	sim := bootSim(t)
	assert.Equal(t, 64, world.Column[square](sim.World).Len())
	assert.Equal(t, 32, world.Column[piece](sim.World).Len())
	_, whiteKing, ok := pieceAt(sim.World, 0, 4)
	require.True(t, ok)
	assert.Equal(t, sideWhite, whiteKing.color)
	assert.Equal(t, kindKing, whiteKing.kind)
	line, ok := world.Read[banner](sim.World)
	require.True(t, ok)
	assert.Equal(t, "Next move: White", line.text)
}

func TestPieceRules(t *testing.T) {
	whitePawn := piece{color: sideWhite, kind: kindPawn, x: 1, y: 4}
	blocker := piece{color: sideBlack, kind: kindPawn, x: 2, y: 4}
	assert.True(t, whitePawn.valid(2, 4, []piece{whitePawn}))
	assert.True(t, whitePawn.valid(3, 4, []piece{whitePawn}))
	assert.False(t, whitePawn.valid(3, 4, []piece{whitePawn, blocker}))
	assert.False(t, whitePawn.valid(2, 4, []piece{whitePawn, blocker}))
	victim := piece{color: sideBlack, kind: kindPawn, x: 2, y: 5}
	assert.True(t, whitePawn.valid(2, 5, []piece{whitePawn, victim}))
	assert.False(t, whitePawn.valid(2, 3, []piece{whitePawn}))

	blackPawn := piece{color: sideBlack, kind: kindPawn, x: 6, y: 3}
	assert.True(t, blackPawn.valid(4, 3, []piece{blackPawn}))
	assert.False(t, blackPawn.valid(7, 3, []piece{blackPawn}))

	knight := piece{color: sideWhite, kind: kindKnight, x: 0, y: 1}
	assert.True(t, knight.valid(2, 2, []piece{knight, blocker}))
	assert.False(t, knight.valid(2, 1, []piece{knight}))

	rook := piece{color: sideWhite, kind: kindRook, x: 0, y: 0}
	pawn := piece{color: sideWhite, kind: kindPawn, x: 1, y: 0}
	assert.False(t, rook.valid(2, 0, []piece{rook, pawn}))
	assert.False(t, rook.valid(0, 0, []piece{rook}))

	bishop := piece{color: sideWhite, kind: kindBishop, x: 0, y: 2}
	assert.True(t, bishop.valid(2, 4, []piece{bishop}))
	block := piece{color: sideBlack, kind: kindPawn, x: 1, y: 3}
	assert.False(t, bishop.valid(2, 4, []piece{bishop, block}))

	king := piece{color: sideWhite, kind: kindKing, x: 0, y: 4}
	assert.True(t, king.valid(1, 4, []piece{king}))
	assert.True(t, king.valid(1, 5, []piece{king}))
	assert.False(t, king.valid(2, 4, []piece{king}))
}

func TestPawnOpeningSwitchesTheTurn(t *testing.T) {
	sim := bootSim(t)
	disableCPU(sim)
	clickSquare(t, sim, 1, 4)
	hold, _ := world.Read[selectedPiece](sim.World)
	require.True(t, hold.ok)
	_, pawn, ok := pieceAt(sim.World, 1, 4)
	require.True(t, ok)
	assert.Equal(t, kindPawn, pawn.kind)

	clickSquare(t, sim, 3, 4)
	_, pawn, ok = pieceAt(sim.World, 3, 4)
	require.True(t, ok)
	assert.Equal(t, sideWhite, pawn.color)
	assert.Equal(t, kindPawn, pawn.kind)
	_, _, still := pieceAt(sim.World, 1, 4)
	assert.False(t, still)
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideBlack, turn.side)
	cleared, _ := world.Read[selectedPiece](sim.World)
	assert.False(t, cleared.ok)
	require.NotEmpty(t, world.Messages[resetSelected](sim.World))

	sim.Frame(t.Context())
	assert.Empty(t, world.Messages[resetSelected](sim.World))
	turn, _ = world.Read[playerTurn](sim.World)
	assert.Equal(t, sideBlack, turn.side)
}

func TestBlockedRookKeepsTheTurn(t *testing.T) {
	sim := bootSim(t)
	clickSquare(t, sim, 0, 0)
	clickSquare(t, sim, 2, 0)
	_, rook, ok := pieceAt(sim.World, 0, 0)
	require.True(t, ok)
	assert.Equal(t, kindRook, rook.kind)
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideWhite, turn.side)
	cleared, _ := world.Read[selectedPiece](sim.World)
	assert.False(t, cleared.ok)
	assertBlackHome(t, sim)
}

func TestKnightJumps(t *testing.T) {
	sim := bootSim(t)
	disableCPU(sim)
	clickSquare(t, sim, 0, 1)
	clickSquare(t, sim, 2, 2)
	_, knight, ok := pieceAt(sim.World, 2, 2)
	require.True(t, ok)
	assert.Equal(t, kindKnight, knight.kind)
	assert.Equal(t, sideWhite, knight.color)
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideBlack, turn.side)
}

func TestCaptureRemovesThePiece(t *testing.T) {
	sim := bootSim(t)
	disableCPU(sim)
	clickSquare(t, sim, 1, 4)
	clickSquare(t, sim, 3, 4)
	clickSquare(t, sim, 6, 3)
	clickSquare(t, sim, 4, 3)
	clickSquare(t, sim, 3, 4)
	clickSquare(t, sim, 4, 3)
	_, pawn, ok := pieceAt(sim.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, sideWhite, pawn.color)
	assert.Equal(t, kindPawn, pawn.kind)
	assert.Equal(t, 31, world.Column[piece](sim.World).Len())
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideBlack, turn.side)
}

func TestKingCaptureEndsTheGame(t *testing.T) {
	sim := bootSim(t)
	var drop []world.Entity
	world.Column[piece](sim.World).Read(func(e world.Entity, p piece) {
		keep := (p.color == sideWhite && p.kind == kindQueen) || (p.color == sideBlack && p.kind == kindKing)
		if !keep {
			drop = append(drop, e)
		}
	})
	for _, e := range drop {
		sim.World.Despawn(e)
	}
	world.Column[piece](sim.World).Each(func(_ world.Entity, p *piece) {
		if p.color == sideBlack && p.kind == kindKing {
			p.x, p.y = 1, 3
		}
	})
	clickSquare(t, sim, 0, 3)
	clickSquare(t, sim, 1, 3)
	assert.Equal(t, 1, world.Column[piece](sim.World).Len())
	_, queen, ok := pieceAt(sim.World, 1, 3)
	require.True(t, ok)
	assert.Equal(t, kindQueen, queen.kind)
	end, ok := world.Read[outcome](sim.World)
	require.True(t, ok)
	assert.True(t, end.over)
	assert.Equal(t, sideWhite, end.winner)
	line, _ := world.Read[banner](sim.World)
	assert.Equal(t, "White won. Thanks for playing!", line.text)

	clickSquare(t, sim, 1, 3)
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideBlack, turn.side)
}

func TestClickOutsideClearsSelection(t *testing.T) {
	sim := bootSim(t)
	clickSquare(t, sim, 1, 4)
	hold, _ := world.Read[selectedPiece](sim.World)
	require.True(t, hold.ok)
	world.Put(sim.World, click{down: true})
	sim.Frame(t.Context())
	hold, _ = world.Read[selectedPiece](sim.World)
	assert.False(t, hold.ok)
	_, pawn, ok := pieceAt(sim.World, 1, 4)
	require.True(t, ok)
	assert.Equal(t, kindPawn, pawn.kind)
}

func TestPointerMovesThePawn(t *testing.T) {
	screen := newScreen(t.Context())
	require.NotNil(t, screen.View())
	for _, sq := range [][2]uint8{{0, 0}, {7, 7}, {1, 4}, {3, 4}} {
		x, y, ok := pickSquare(screen.size, squareCenter(screen.size, sq[0], sq[1]))
		require.True(t, ok)
		assert.Equal(t, sq[0], x)
		assert.Equal(t, sq[1], y)
	}

	from := squareCenter(image.Pt(640, 720), 1, 4)
	to := squareCenter(image.Pt(640, 720), 3, 4)
	_, cmd := screen.Update(window.Pointer{Pos: from, Button: 1, Pressed: true})
	assert.Nil(t, cmd)
	_, cmd = screen.Update(window.Pointer{Pos: to, Button: 1, Pressed: true})
	assert.Nil(t, cmd)
	_, pawn, ok := pieceAt(screen.sim.World, 3, 4)
	require.True(t, ok)
	assert.Equal(t, sideWhite, pawn.color)
	assert.Equal(t, kindPawn, pawn.kind)
	_, reply, ok := pieceAt(screen.sim.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, sideBlack, reply.color)
	assert.Equal(t, kindPawn, reply.kind)
	line, _ := world.Read[banner](screen.sim.World)
	assert.Equal(t, "Next move: White", line.text)
}

func TestCPUPluginComposesAheadOfTheBoard(t *testing.T) {
	sim := world.New()
	sim.Plugin(cpuPlugin)
	sim.Plugin(piecesPlugin)
	sim.Plugin(boardPlugin)
	sim.Frame(t.Context())
	clickSquare(t, sim, 1, 4)
	clickSquare(t, sim, 3, 4)
	_, black, ok := pieceAt(sim.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, piece{color: sideBlack, kind: kindPawn, x: 4, y: 3}, black)
}

func TestCPUAnswersTheOpening(t *testing.T) {
	sim := bootSim(t)
	clickSquare(t, sim, 1, 4)
	clickSquare(t, sim, 3, 4)
	_, white, ok := pieceAt(sim.World, 3, 4)
	require.True(t, ok)
	assert.Equal(t, piece{color: sideWhite, kind: kindPawn, x: 3, y: 4}, white)
	_, black, ok := pieceAt(sim.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, piece{color: sideBlack, kind: kindPawn, x: 4, y: 3}, black)
	_, _, still := pieceAt(sim.World, 6, 3)
	assert.False(t, still)
	assert.Equal(t, 32, world.Column[piece](sim.World).Len())
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideWhite, turn.side)
	line, _ := world.Read[banner](sim.World)
	assert.Equal(t, "Next move: White", line.text)

	again := bootSim(t)
	clickSquare(t, again, 1, 4)
	clickSquare(t, again, 3, 4)
	_, other, ok := pieceAt(again.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, black, other)

	sim.Frame(t.Context())
	_, stayed, ok := pieceAt(sim.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, black, stayed)
	turn, _ = world.Read[playerTurn](sim.World)
	assert.Equal(t, sideWhite, turn.side)
	assert.Equal(t, 32, world.Column[piece](sim.World).Len())
}

func TestCPUTakesTheKing(t *testing.T) {
	sim := bootSim(t)
	var drop []world.Entity
	world.Column[piece](sim.World).Read(func(e world.Entity, p piece) {
		keep := (p.color == sideBlack && p.kind == kindQueen) || (p.color == sideWhite && p.kind == kindKing)
		if !keep {
			drop = append(drop, e)
		}
	})
	for _, e := range drop {
		sim.World.Despawn(e)
	}
	world.Column[piece](sim.World).Each(func(_ world.Entity, p *piece) {
		switch p.kind {
		case kindQueen:
			p.x, p.y = 2, 4
		case kindKing:
			p.x, p.y = 0, 4
		}
	})
	world.Put(sim.World, playerTurn{side: sideBlack})
	sim.Frame(t.Context())
	end, ok := world.Read[outcome](sim.World)
	require.True(t, ok)
	assert.True(t, end.over)
	assert.Equal(t, sideBlack, end.winner)
	assert.Equal(t, 1, world.Column[piece](sim.World).Len())
	_, queen, ok := pieceAt(sim.World, 0, 4)
	require.True(t, ok)
	assert.Equal(t, kindQueen, queen.kind)
	line, _ := world.Read[banner](sim.World)
	assert.Equal(t, "Black won. Thanks for playing!", line.text)
}

func TestReplyLeavesWithTheCapture(t *testing.T) {
	sim := bootSim(t)
	var drop []world.Entity
	world.Column[piece](sim.World).Read(func(e world.Entity, p piece) {
		keep := p.kind == kindKing || (p.color == sideWhite && p.kind == kindPawn && p.y == 4) || (p.color == sideBlack && p.kind == kindPawn && p.y == 3)
		if !keep {
			drop = append(drop, e)
		}
	})
	for _, e := range drop {
		sim.World.Despawn(e)
	}
	world.Column[piece](sim.World).Each(func(_ world.Entity, p *piece) {
		switch {
		case p.color == sideWhite && p.kind == kindPawn:
			p.x, p.y = 3, 4
		case p.color == sideBlack && p.kind == kindPawn:
			p.x, p.y = 4, 3
		}
	})
	clickSquare(t, sim, 3, 4)
	clickSquare(t, sim, 4, 3)
	assert.Equal(t, 3, world.Column[piece](sim.World).Len())
	_, pawn, ok := pieceAt(sim.World, 4, 3)
	require.True(t, ok)
	assert.Equal(t, sideWhite, pawn.color)
	assert.Equal(t, kindPawn, pawn.kind)
	_, king, ok := pieceAt(sim.World, 6, 3)
	require.True(t, ok)
	assert.Equal(t, sideBlack, king.color)
	assert.Equal(t, kindKing, king.kind)
	turn, _ := world.Read[playerTurn](sim.World)
	assert.Equal(t, sideWhite, turn.side)
}

func TestSceneIsTheBoard(t *testing.T) {
	screen := newScreen(t.Context())
	require.NotNil(t, screen.frame)
	var light, dark, ivory, ebony int
	b := screen.frame.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := screen.frame.RGBAAt(x, y)
			switch {
			case c.R > 180 && c.B > 160 && c.G > 150 && int(c.R)-int(c.B) < 40:
				light++
			case c.R < 20 && c.B > 8 && c.B+6 >= c.R:
				dark++
			case c.R > 170 && c.G > 140 && int(c.R) > int(c.B)+40:
				ivory++
			case c.R > 28 && c.R < 110 && absInt(int(c.R)-int(c.G)) < 16 && absInt(int(c.R)-int(c.B)) < 22:
				ebony++
			}
		}
	}
	at := func(x, y uint8) (r, g, b int) {
		t.Helper()
		p := squareCenter(screen.size, x, y)
		c := screen.frame.RGBAAt(p.X, p.Y)
		return int(c.R), int(c.G), int(c.B)
	}
	// (3,4) is an empty pale square. (4,4) is an empty near-black square.
	lr, _, lb := at(3, 4)
	assert.Greater(t, lr, 200)
	assert.Greater(t, lb, 180)
	dr, _, db := at(4, 4)
	assert.Less(t, dr, 20)
	assert.Greater(t, db, 15)
	assert.Greater(t, light, 4000)
	assert.Greater(t, dark, 4000)
	assert.Greater(t, ivory, 400)
	assert.Greater(t, ebony, 200)

	images, texts, rasters := 0, 0, 0
	walkNodes(screen.View(), &images, &texts, &rasters)
	assert.Equal(t, 1, images)
	assert.Equal(t, 1, texts)
	assert.Equal(t, 0, rasters)
}

func TestHoverRecolorsTheSquare(t *testing.T) {
	screen := newScreen(t.Context())
	require.True(t, screen.Consume())
	require.Equal(t, 1, screen.fullPaints)

	pt := squareCenter(screen.size, 4, 4)
	_, _ = screen.Update(window.Pointer{Pos: pt})
	require.True(t, screen.Consume())
	assert.Equal(t, 1, screen.fullPaints)
	c := screen.frame.RGBAAt(pt.X, pt.Y)
	assert.Greater(t, int(c.R), 160)
	assert.Less(t, int(c.G), 120)
	assert.Less(t, int(c.B), 120)

	nudge := pt.Add(image.Pt(3, 2))
	x, y, ok := pickSquare(screen.size, nudge)
	require.True(t, ok)
	require.Equal(t, uint8(4), x)
	require.Equal(t, uint8(4), y)
	_, _ = screen.Update(window.Pointer{Pos: nudge})
	assert.False(t, screen.Consume())
	assert.Equal(t, 1, screen.fullPaints)

	_, _ = screen.Update(gui.TickMsg{Elapsed: time.Second, Period: time.Second / 60, Size: screen.size})
	assert.False(t, screen.Consume())
	assert.Equal(t, 1, screen.fullPaints)

	var found bool
	world.Query[piece](screen.sim.World).Each(func(_ world.Entity, p *piece) {
		if found || p.color != sideWhite || p.kind != kindPawn || p.y != 4 {
			return
		}
		p.x = 3
		found = true
	})
	require.True(t, found)
	_, _ = screen.Update(gui.TickMsg{Elapsed: time.Second + 500*time.Millisecond, Period: time.Second / 60, Size: screen.size})
	assert.True(t, screen.Consume())
	assert.Equal(t, 2, screen.fullPaints)
}

func TestPieceSlidesTowardTheSquare(t *testing.T) {
	sim := bootSim(t)
	var id world.Entity
	var found bool
	world.Query[piece](sim.World).Each(func(e world.Entity, p *piece) {
		if found || p.color != sideWhite || p.kind != kindPawn || p.y != 4 {
			return
		}
		p.x = 3
		id = e
		found = true
	})
	require.True(t, found)
	sim.Step(t.Context(), 0.5)
	at, ok := world.Query[pose](sim.World).Get(id)
	require.True(t, ok)
	assert.InDelta(t, 1.5, float64(at.x), 0.05)
	assert.InDelta(t, 4, float64(at.z), 0.02)
}

func assertBlackHome(t *testing.T, sim *world.Sim) {
	t.Helper()
	world.Column[piece](sim.World).Read(func(_ world.Entity, p piece) {
		if p.color != sideBlack {
			return
		}
		home := uint8(7)
		if p.kind == kindPawn {
			home = 6
		}
		assert.Equal(t, home, p.x)
	})
}

func TestBoardStaysBelowTheBar(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	const width, height, bar = 160, 200, 28
	host, err := window.Open(t.Context(), window.Config{Width: width, Height: height, Period: time.Hour})
	require.NoError(t, err)
	test.CloseOnCleanup(t, host)
	setter, ok := host.(interface{ SetDead([]image.Rectangle) })
	require.True(t, ok)
	setter.SetDead([]image.Rectangle{image.Rect(0, 0, width, bar)})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- gui.Run(ctx, host, ndarray.CPU, newScreen(ctx)) }()
	light := squareCenter(image.Pt(width, height-bar), 3, 4)
	require.Eventually(t, func() bool {
		return host.Front().RGBAAt(8, 8) == color.RGBA{0, 0, 0, 255} &&
			host.Front().RGBAAt(light.X, light.Y+bar).R > 180
	}, 2*time.Second, 10*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
}

func walkNodes(n gui.Node, images, texts, rasters *int) {
	switch node := n.(type) {
	case *inset:
		walkNodes(node.screen.view(), images, texts, rasters)
	case *gui.Hint:
		walkNodes(node.Child, images, texts, rasters)
	case *gui.Stack:
		for _, child := range node.Children {
			walkNodes(child, images, texts, rasters)
		}
	case *gui.Positioned:
		walkNodes(node.Child, images, texts, rasters)
	case *gui.Box:
		if node.Child != nil {
			walkNodes(node.Child, images, texts, rasters)
		}
	case *gui.Image:
		*images++
	case *gui.Text:
		*texts++
	case *gui.Raster:
		*rasters++
	}
}
