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
	assert.Equal(t, []int{7}, early)
	assert.Empty(t, late)

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
	edited := col.Mut(e)
	require.NotNil(t, edited)
	assert.False(t, col.Changed(e))
	*edited = 5
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
	edited := Query[int](w).Mut(keep)
	require.NotNil(t, edited)
	assert.Empty(t, Changed(Query[int](w)).All())
	*edited = 4
	assert.Equal(t, []int{4}, Changed(Query[int](w)).All())
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

func TestRemoveThenInsertKeepsTheValue(t *testing.T) {
	sim := New()
	e := Spawn(sim.World, 1)
	sim.System(Update, func(_ context.Context, w *World) {
		Column[int](w).Remove(e)
		Column[int](w).Insert(e, 5)
		got, ok := Column[int](w).Get(e)
		require.True(t, ok)
		assert.Equal(t, 5, got)
	})
	var after int
	var ok bool
	sim.System(PostUpdate, func(_ context.Context, w *World) {
		after, ok = Column[int](w).Get(e)
	})
	sim.Frame(t.Context())
	require.True(t, ok)
	assert.Equal(t, 5, after)
}

func TestInsertThenRemoveDropsTheValue(t *testing.T) {
	sim := New()
	e := Spawn(sim.World, 1)
	sim.System(Update, func(_ context.Context, w *World) {
		Column[int](w).Insert(e, 5)
		Column[int](w).Remove(e)
	})
	sim.System(PostUpdate, func(_ context.Context, w *World) {
		_, ok := Column[int](w).Get(e)
		assert.False(t, ok)
	})
	sim.Frame(t.Context())
}

func TestEachDoesNotDirtyARead(t *testing.T) {
	sim := New()
	e := Spawn(sim.World, 1)
	sim.System(Update, func(_ context.Context, w *World) {
		Column[int](w).Each(func(Entity, *int) {})
	})
	sim.System(PostUpdate, func(_ context.Context, w *World) {
		assert.False(t, Column[int](w).Changed(e))
	})
	sim.Frame(t.Context())
}

func TestChangeReachesAnEarlierSystemNextFrame(t *testing.T) {
	sim := New()
	e := Spawn(sim.World, 0)
	var n int
	var early bool
	writer := func(_ context.Context, w *World) {
		n++
		if n != 1 {
			return
		}
		got := Column[int](w).Mut(e)
		*got = 1
	}
	reader := func(_ context.Context, w *World) {
		early = Column[int](w).Changed(e)
	}
	sim.System(Update, writer)
	sim.System(Update, reader).Before(writer)

	sim.Frame(t.Context())
	assert.False(t, early)
	sim.Frame(t.Context())
	assert.True(t, early)
	sim.Frame(t.Context())
	assert.False(t, early)
}

func TestSendBeforeFrameReachesSystems(t *testing.T) {
	sim := New()
	var got []int
	sim.System(Update, func(_ context.Context, w *World) {
		got = Messages[int](w)
	})
	Send(sim.World, 4)
	sim.Frame(t.Context())
	assert.Equal(t, []int{4}, got)
	sim.Frame(t.Context())
	assert.Empty(t, got)
}

func TestPutKeepsTheMutPointer(t *testing.T) {
	w := newWorld()
	Put(w, 1)
	p := Mut[int](w)
	require.NotNil(t, p)
	Put(w, 2)
	assert.Equal(t, 2, *p)
}

func TestDeferredDeferRunsAfterTheBatch(t *testing.T) {
	sim := New()
	var got []int
	sim.System(Update, func(_ context.Context, w *World) {
		w.Defer(func() {
			got = append(got, 1)
			w.Defer(func() { got = append(got, 2) })
		})
		w.Defer(func() { got = append(got, 3) })
	})
	sim.Frame(t.Context())
	assert.Equal(t, []int{1, 3, 2}, got)
}

func TestNilContextPanics(t *testing.T) {
	assert.Panics(t, func() { New().Frame(nil) })
}

func TestBadScheduleDoesNotRun(t *testing.T) {
	sim := New()
	ran := false
	sim.System(Startup, func(context.Context, *World) { ran = true })
	sim.System(Last, func(context.Context, *World) {}).After(func(context.Context, *World) {})
	assert.Panics(t, func() { sim.Frame(t.Context()) })
	assert.False(t, ran)
	assert.Equal(t, uint64(0), sim.World.Tick())
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
