package world

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityRecyclesGeneration(t *testing.T) {
	w := newWorld()
	first := w.Spawn()
	w.Despawn(first)
	second := w.Spawn()

	assert.Equal(t, first.Index, second.Index)
	assert.NotEqual(t, first.Gen, second.Gen)
	assert.False(t, w.Alive(first))
	assert.True(t, w.Alive(second))

	col := Column[int](w)
	col.Insert(first, 1)
	_, ok := col.Get(first)
	assert.False(t, ok)
	col.Insert(second, 9)
	got, ok := col.Get(second)
	require.True(t, ok)
	assert.Equal(t, 9, got)
}

func TestDespawnWaitsUntilSystemEnds(t *testing.T) {
	sim := New()
	w := sim.World
	e := w.Spawn()
	col := Column[int](w)
	col.Insert(e, 1)
	var seen int

	sim.System(Update, func(context.Context, *World) {
		col.Each(func(id Entity, v *int) {
			seen++
			w.Despawn(id)
			assert.True(t, w.Alive(id))
			got, ok := col.Get(id)
			assert.True(t, ok)
			assert.Equal(t, 1, got)
			*v = 4
		})
		assert.True(t, w.Alive(e))
		assert.Equal(t, 1, col.Len())
	})
	sim.System(PostUpdate, func(context.Context, *World) {
		assert.False(t, w.Alive(e))
		assert.Equal(t, 0, col.Len())
	})

	sim.Frame(t.Context())
	assert.Equal(t, 1, seen)
	assert.False(t, w.Alive(e))
}

func TestEachSkipsRowsInsertedDuringTheWalk(t *testing.T) {
	sim := New()
	w := sim.World
	col := Column[int](w)
	col.Insert(w.Spawn(), 1)
	var seen int

	sim.System(Update, func(context.Context, *World) {
		col.Each(func(Entity, *int) {
			seen++
			col.Insert(w.Spawn(), 2)
		})
	})
	sim.Frame(t.Context())
	assert.Equal(t, 1, seen)
	assert.Equal(t, 2, col.Len())
}

func TestScheduleOrderAndMessages(t *testing.T) {
	sim := New()
	var early, late []int
	sim.System(First, func(_ context.Context, w *World) {
		early = append([]int(nil), Messages[int](w)...)
	})
	sim.System(Update, func(_ context.Context, w *World) {
		if w.Tick() == 1 {
			Send(w, 7)
		}
	})
	sim.System(PostUpdate, func(_ context.Context, w *World) {
		late = append([]int(nil), Messages[int](w)...)
	})

	sim.Frame(t.Context())
	assert.Empty(t, early)
	assert.Equal(t, []int{7}, late)

	sim.Frame(t.Context())
	assert.Empty(t, early)
	assert.Empty(t, late)
}

func TestStartupRunsOnce(t *testing.T) {
	sim := New()
	var n int
	sim.System(Startup, func(context.Context, *World) { n++ })
	sim.Frame(t.Context())
	sim.Frame(t.Context())
	assert.Equal(t, 1, n)
}

func TestPluginJoinAdvancesMotion(t *testing.T) {
	type place struct{ X int }
	type speed struct{ X int }

	sim := New()
	sim.Plugin(func(s *Sim) {
		s.System(Startup, func(_ context.Context, w *World) {
			e := w.Spawn()
			Column[place](w).Insert(e, place{X: 1})
			Column[speed](w).Insert(e, speed{X: 2})
			w.Spawn()
		})
		s.System(Update, func(_ context.Context, w *World) {
			Join(Column[place](w), Column[speed](w), func(_ Entity, p *place, v *speed) {
				p.X += v.X
			})
		})
	})

	sim.Frame(t.Context())
	sim.Frame(t.Context())

	var got int
	Column[place](sim.World).Read(func(_ Entity, p place) { got = p.X })
	assert.Equal(t, 5, got)
	assert.Equal(t, 1, Column[place](sim.World).Len())
	assert.True(t, Column[place](sim.World).Changed(entityWith(t, Column[place](sim.World))))
}

func TestChangedStamp(t *testing.T) {
	sim := New()
	w := sim.World
	e := w.Spawn()
	col := Column[int](w)
	col.Insert(e, 3)
	assert.False(t, col.Changed(e))

	sim.Frame(t.Context())
	assert.False(t, col.Changed(e))
	col.Insert(e, 4)
	assert.True(t, col.Changed(e))

	sim.Frame(t.Context())
	assert.False(t, col.Changed(e))
	assert.Equal(t, 4, *col.Mut(e))
	assert.True(t, col.Changed(e))

	type clock struct{ N int }
	Put(w, clock{N: 1})
	assert.True(t, Written[clock](w))
	sim.Frame(t.Context())
	assert.False(t, Written[clock](w))
	got, ok := Read[clock](w)
	require.True(t, ok)
	assert.Equal(t, 1, got.N)
	assert.False(t, Written[clock](w))
	Mut[clock](w).N = 2
	assert.True(t, Written[clock](w))
}

func TestQueryWithoutAndChanged(t *testing.T) {
	type mark struct{}
	sim := New()
	w := sim.World
	keep := Spawn(w, 1)
	drop := Spawn(w, 2)
	Insert(w, drop, mark{})

	var got []int
	Without[int, mark](Query[int](w)).Read(func(_ Entity, v int) {
		got = append(got, v)
	})
	assert.Equal(t, []int{1}, got)
	_, ok := Without[int, mark](Query[int](w)).Get(drop)
	assert.False(t, ok)
	gotValue, ok := Without[int, mark](Query[int](w)).Get(keep)
	require.True(t, ok)
	assert.Equal(t, 1, gotValue)

	sim.Frame(t.Context())
	assert.Empty(t, Changed(Query[int](w)).All())
	require.NotNil(t, Query[int](w).Mut(keep))
	assert.Equal(t, []int{1}, Changed(Query[int](w)).All())
}

func TestInitKeepsTheFirstValue(t *testing.T) {
	w := newWorld()
	Init(w, 1)
	Init(w, 2)
	got, ok := Read[int](w)
	require.True(t, ok)
	assert.Equal(t, 1, got)
}

func TestSpawn2StoresBothColumns(t *testing.T) {
	type who struct{ Text string }
	w := newWorld()
	e := Spawn2(w, 7, who{Text: "Elaina"})
	got, ok := Column[int](w).Get(e)
	require.True(t, ok)
	assert.Equal(t, 7, got)
	name, ok := Query[who](w).Get(e)
	require.True(t, ok)
	assert.Equal(t, "Elaina", name.Text)
	_, _, found := With[who, int](Query[who](w)).First(nil)
	assert.True(t, found)
}

func TestChainInsertsBetweenPlugins(t *testing.T) {
	for _, flip := range []bool{false, true} {
		sim := New()
		var got []string
		a := func(context.Context, *World) { got = append(got, "a") }
		b := func(context.Context, *World) { got = append(got, "b") }
		c := func(context.Context, *World) { got = append(got, "c") }
		board := func(s *Sim) { s.Chain(Update, a, c) }
		extra := func(s *Sim) { s.System(Update, b).After(a).Before(c) }
		if flip {
			sim.Plugin(extra)
			sim.Plugin(board)
		} else {
			sim.Plugin(board)
			sim.Plugin(extra)
		}
		sim.Frame(t.Context())
		assert.Equal(t, []string{"a", "b", "c"}, got)
	}
}

func TestStepWritesTimeInsideTheFrame(t *testing.T) {
	sim := New()
	var delta, elapsed float64
	var written bool
	sim.System(Update, func(_ context.Context, w *World) {
		clock, ok := Read[Time](w)
		require.True(t, ok)
		delta, elapsed = clock.Delta, clock.Elapsed
		written = Written[Time](w)
	})
	sim.Step(t.Context(), 0.5)
	assert.Equal(t, 0.5, delta)
	assert.Equal(t, 0.5, elapsed)
	assert.True(t, written)
	sim.Step(t.Context(), -1)
	assert.Equal(t, 0.0, delta)
	assert.Equal(t, 0.5, elapsed)
}

func TestUnknownOrderAndCyclePanic(t *testing.T) {
	sim := New()
	sim.System(Update, func(context.Context, *World) {}).After(func(context.Context, *World) {})
	assert.Panics(t, func() { sim.Frame(t.Context()) })

	cycled := New()
	a := func(context.Context, *World) {}
	b := func(context.Context, *World) {}
	cycled.System(Update, a).After(b)
	cycled.System(Update, b).After(a)
	assert.Panics(t, func() { cycled.Frame(t.Context()) })
}

func TestFrameStopsWhenContextIsCanceled(t *testing.T) {
	sim := New()
	ran := 0
	ctx, cancel := context.WithCancel(t.Context())
	sim.System(Update, func(context.Context, *World) {
		ran++
		cancel()
	})
	sim.System(PostUpdate, func(context.Context, *World) { ran += 10 })
	sim.Frame(ctx)
	assert.Equal(t, 1, ran)

	blocked, stop := context.WithCancel(t.Context())
	stop()
	sim.Frame(blocked)
	assert.Equal(t, 1, ran)
}

func TestResourceTypesDoNotCollide(t *testing.T) {
	w := newWorld()
	Put[int](w, 1)
	Put[int64](w, 2)
	e := w.Spawn()
	Column[int](w).Insert(e, 9)
	a, ok := Read[int](w)
	require.True(t, ok)
	b, ok := Read[int64](w)
	require.True(t, ok)
	assert.Equal(t, 1, a)
	assert.Equal(t, int64(2), b)
	got, ok := Column[int](w).Get(e)
	require.True(t, ok)
	assert.Equal(t, 9, got)
	assert.Nil(t, Mut[string](w))
}

func entityWith[T any](t *testing.T, col *Table[T]) Entity {
	t.Helper()
	var id Entity
	var found bool
	col.Read(func(e Entity, _ T) {
		id = e
		found = true
	})
	require.True(t, found)
	return id
}
