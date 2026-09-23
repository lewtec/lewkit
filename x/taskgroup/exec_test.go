package taskgroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nop(_ context.Context, _ *Status) error { return nil }

func newPinned(t *testing.T) *Session {
	t.Helper()
	s, _ := New(t.Context(), DefaultLimits())
	t.Cleanup(func() {
		s.Cancel(nil)
		_ = s.Wait()
	})
	return s
}

func TestPinDepsSelfCycle(t *testing.T) {
	s := newPinned(t)
	s.mu.Lock()
	id := s.alloc(s.root, "loop", CPU, nop, false, s.ctx)
	s.pinDeps(id, []ID{id})
	st := State(s.slots[id].state.Load())
	err := s.slots[id].err
	s.mu.Unlock()
	assert.Equal(t, Failed, st)
	require.ErrorIs(t, err, ErrCycle)
}

func TestPinDepsTwoNodeCycle(t *testing.T) {
	s := newPinned(t)
	s.mu.Lock()
	a := s.alloc(s.root, "a", CPU, nop, false, s.ctx)
	b := s.alloc(s.root, "b", CPU, nop, false, s.ctx)
	s.pinDeps(b, []ID{a})
	require.Equal(t, Pending, State(s.slots[b].state.Load()))
	s.pinDeps(a, []ID{b})
	st := State(s.slots[a].state.Load())
	err := s.slots[a].err
	s.mu.Unlock()
	assert.Equal(t, Failed, st)
	require.ErrorIs(t, err, ErrCycle)
}

func TestPinDepsThreeNodeCycle(t *testing.T) {
	s := newPinned(t)
	s.mu.Lock()
	a := s.alloc(s.root, "a", CPU, nop, false, s.ctx)
	b := s.alloc(s.root, "b", CPU, nop, false, s.ctx)
	c := s.alloc(s.root, "c", CPU, nop, false, s.ctx)
	s.pinDeps(b, []ID{a})
	s.pinDeps(c, []ID{b})
	s.pinDeps(a, []ID{c})
	err := s.slots[a].err
	s.mu.Unlock()
	require.ErrorIs(t, err, ErrCycle)
	assert.Equal(t, Failed, State(s.slots[a].state.Load()))
}

func TestPinDepsDiamondIsNotACycle(t *testing.T) {
	s := newPinned(t)
	s.mu.Lock()
	root := s.alloc(s.root, "root", CPU, nop, false, s.ctx)
	left := s.alloc(s.root, "left", CPU, nop, false, s.ctx)
	right := s.alloc(s.root, "right", CPU, nop, false, s.ctx)
	join := s.alloc(s.root, "join", CPU, nop, false, s.ctx)
	s.pinDeps(left, []ID{root})
	s.pinDeps(right, []ID{root})
	s.pinDeps(join, []ID{left, right})
	s.mu.Unlock()
	assert.Equal(t, Pending, State(s.slots[join].state.Load()))
	assert.Nil(t, s.slots[join].err)
	assert.Equal(t, uint32(2), s.slots[join].remainingDeps)
}
