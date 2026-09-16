package taskgroup

import (
	"context"
	"fmt"
)

// Go schedules a task under the current node (or the session root).
// deps are node IDs returned by earlier Go calls.
func Go(ctx context.Context, name string, pool PoolKind, fn func(context.Context, *Status) error, deps ...ID) ID {
	return MustFromContext(ctx).Go(ctx, name, pool, fn, deps...)
}

// Go schedules a task under the node carried by ctx, or the root.
func (s *Session) Go(ctx context.Context, name string, pool PoolKind, fn func(context.Context, *Status) error, deps ...ID) ID {
	return s.goTask(ctx, name, pool, fn, false, deps...)
}

func (s *Session) goTask(ctx context.Context, name string, pool PoolKind, fn func(context.Context, *Status) error, isolate bool, deps ...ID) ID {
	parent := taskFromContext(ctx)
	if parent == 0 {
		parent = s.root
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.alloc(parent, name, pool, fn, isolate)
	if fn == nil {
		s.finishLocked(id, ErrNilFn)
		return id
	}
	s.pinDeps(id, deps)
	if State(s.slots[id].state.Load()) == Pending && s.slots[id].remainingDeps == 0 {
		s.enqueueLocked(id)
	}
	return id
}

func (s *Session) pinDeps(id ID, deps []ID) {
	t := s.slots[id]
	for _, d := range deps {
		if d == 0 || int(d) >= len(s.slots) || s.slots[d] == nil || s.slots[d].id != d {
			s.finishLocked(id, fmt.Errorf("taskgroup: %w %d", ErrUnknownDependency, d))
			return
		}
		dep := s.slots[d]
		st := State(dep.state.Load())
		if st == Failed {
			s.finishLocked(id, fmt.Errorf("taskgroup: %w: %q: %w", ErrDependencyFailed, dep.name, dep.err))
			return
		}
		if st == Done {
			continue
		}
		if d == id || s.reaches(id, d) {
			s.finishLocked(id, fmt.Errorf("taskgroup: %w", ErrCycle))
			return
		}
		t.remainingDeps++
		dep.waiters = append(dep.waiters, id)
	}
}

// reaches reports whether to is reachable from from by following waiters
// (tasks that depend on from). Caller holds s.mu.
func (s *Session) reaches(from, to ID) bool {
	seen := make(map[ID]struct{})
	var walk func(ID) bool
	walk = func(id ID) bool {
		if _, ok := seen[id]; ok {
			return false
		}
		seen[id] = struct{}{}
		t := s.slots[id]
		if t == nil {
			return false
		}
		for _, w := range t.waiters {
			if w == to || walk(w) {
				return true
			}
		}
		return false
	}
	return walk(from)
}

func (s *Session) enqueueLocked(id ID) {
	t := s.slots[id]
	if t.pool == Control {
		go s.run(id)
		return
	}
	if q := s.ready[t.pool]; q != nil {
		q.push(id)
	}
}

func (s *Session) run(id ID) {
	s.mu.Lock()
	t := s.slots[id]
	s.mu.Unlock()
	if t == nil || !t.state.CompareAndSwap(uint32(Pending), uint32(Running)) {
		return
	}
	s.mu.Lock()
	s.moveLiveFront(id)
	s.mu.Unlock()

	ctx := t.ctx
	if ctx == nil {
		ctx = s.ctx
	}
	ctx = context.WithValue(ctx, sessionKey{}, s)
	ctx = context.WithValue(ctx, taskKey{}, id)

	var err error
	if ctx.Err() != nil {
		err = context.Cause(ctx)
	} else {
		err = t.fn(ctx, &Status{t: t})
	}
	s.mu.Lock()
	s.finishLocked(id, err)
	s.mu.Unlock()
}

func (s *Session) finishLocked(id ID, err error) {
	t := s.slots[id]
	st := State(t.state.Load())
	if st == Done || st == Failed {
		return
	}
	if err != nil {
		t.err = err
		t.state.Store(uint32(Failed))
	} else {
		t.state.Store(uint32(Done))
	}
	if t.parent != 0 && t.liveN.Load() == 0 {
		s.unlinkLive(id)
		s.pruneIdleAncestors(t.parent)
	}
	if err != nil && !t.isolate {
		if anc := s.isolateOf(id); anc != 0 {
			a := s.slots[anc]
			if a.err == nil {
				a.err = err
			}
			if a.cancel != nil {
				a.cancel(err)
			}
			s.failSubtree(anc)
		} else if s.err == nil {
			s.recordError(err)
		}
	}
	for _, w := range t.waiters {
		dep := s.slots[w]
		if dep.remainingDeps > 0 {
			dep.remainingDeps--
		}
		if err != nil {
			s.finishLocked(w, fmt.Errorf("taskgroup: %w: %q: %w", ErrDependencyFailed, t.name, err))
			continue
		}
		if dep.remainingDeps == 0 && State(dep.state.Load()) == Pending {
			s.enqueueLocked(w)
		}
	}
	t.waiters = nil
	if t.name != "" && s.latestByDesc[t.name] == id {
		delete(s.latestByDesc, t.name)
	}
	s.tasks.Done()
	s.cond.Broadcast()
}

func (s *Session) pruneIdleAncestors(id ID) {
	for id != 0 && id != s.root {
		t := s.slots[id]
		st := State(t.state.Load())
		if st != Done && st != Failed {
			return
		}
		if t.liveN.Load() > 0 {
			return
		}
		s.unlinkLive(id)
		id = t.parent
	}
}

func (s *Session) isolateOf(id ID) ID {
	for p := s.slots[id].parent; p != 0 && p != s.root; p = s.slots[p].parent {
		if s.slots[p].isolate {
			return p
		}
	}
	if s.slots[id].isolate {
		return id
	}
	return 0
}

func (s *Session) recordError(err error) {
	if s.err != nil {
		return
	}
	s.err = err
	s.cancel(err)
	s.failOutsideIsolates(s.root)
}

func (s *Session) failOutsideIsolates(id ID) {
	t := s.slots[id]
	for c := t.firstChild; c != 0; {
		next := s.slots[c].nextSib
		if s.slots[c].isolate {
			c = next
			continue
		}
		s.failOutsideIsolates(c)
		c = next
	}
	if id != s.root && State(t.state.Load()) == Pending {
		s.finishLocked(id, context.Cause(s.ctx))
	}
}

func (s *Session) failSubtree(id ID) {
	t := s.slots[id]
	for c := t.firstChild; c != 0; {
		next := s.slots[c].nextSib
		s.failSubtree(c)
		c = next
	}
	if id != s.root && State(t.state.Load()) == Pending {
		err := t.err
		if err == nil {
			if p := t.parent; p != 0 && s.slots[p].err != nil {
				err = s.slots[p].err
			} else {
				err = context.Cause(t.ctx)
			}
		}
		s.finishLocked(id, err)
	}
}

func (s *Session) waitLive(ctx context.Context, id ID) error {
	stop := context.AfterFunc(ctx, func() { s.cond.Broadcast() })
	defer stop()

	s.mu.Lock()
	defer s.mu.Unlock()
	for s.slots[id].liveN.Load() > 0 {
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		s.cond.Wait()
	}
	if s.slots[id].err != nil {
		return s.slots[id].err
	}
	return s.err
}

func (s *Session) waitTask(ctx context.Context, id ID) error {
	stop := context.AfterFunc(ctx, func() { s.cond.Broadcast() })
	defer stop()

	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		st := State(s.slots[id].state.Load())
		if st == Done || st == Failed {
			return s.slots[id].err
		}
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		s.cond.Wait()
	}
}
