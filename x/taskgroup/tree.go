package taskgroup

import "context"

// List walks live children from the root until it has n rows.
// Finished leaves leave the live list; a finished parent stays until
// its subtree is idle. List(n) is O(n), not O(scheduled).
func List(ctx context.Context, n int) []Node {
	if s := FromContext(ctx); s != nil {
		return s.List(n)
	}
	return nil
}

// List walks the live tree until it has n rows. Empty-name isolate
// boundaries are skipped but their children are visited.
func (s *Session) List(n int) []Node {
	if n <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Node, 0, n)
	s.walkLive(s.root, n, &out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Session) walkLive(id ID, n int, out *[]Node) {
	if len(*out) >= n {
		return
	}
	if id != s.root && s.slots[id].name != "" {
		*out = append(*out, s.snapshot(id))
		if len(*out) >= n {
			return
		}
	}
	for c := s.slots[id].firstLive; c != 0 && len(*out) < n; c = s.slots[c].nextLive {
		s.walkLive(c, n, out)
	}
}

func (s *Session) snapshot(id ID) Node {
	t := s.slots[id]
	n := Node{
		ID:           id,
		Parent:       t.parent,
		Name:         t.name,
		Pool:         t.pool,
		State:        State(t.state.Load()),
		Current:      t.current.Load(),
		Total:        t.total.Load(),
		LiveChildren: int(t.liveN.Load()),
	}
	if n.Parent == s.root {
		n.Parent = 0
	}
	if p := t.message.Load(); p != nil {
		n.Message = *p
	}
	return n
}

func (s *Session) alloc(parent ID, name string, pool PoolKind, fn func(context.Context, *Status) error, isolate bool) ID {
	id := ID(len(s.slots))
	t := &task{
		id:      id,
		name:    name,
		pool:    pool,
		fn:      fn,
		isolate: isolate,
	}
	t.total.Store(-1)
	t.state.Store(uint32(Pending))

	pctx := s.ctx
	if parent != 0 {
		if pc := s.slots[parent].ctx; pc != nil {
			pctx = pc
		}
	}
	if isolate {
		t.ctx, t.cancel = context.WithCancelCause(pctx)
	} else {
		t.ctx = pctx
	}

	s.slots = append(s.slots, t)
	s.linkChild(parent, id)
	s.linkLive(parent, id)
	if name != "" {
		s.latestByDesc[name] = id
	}
	s.tasks.Add(1)
	return id
}

func (s *Session) linkChild(parent, child ID) {
	p := s.slots[parent]
	c := s.slots[child]
	c.parent = parent
	c.prevSib = p.lastChild
	if p.lastChild != 0 {
		s.slots[p.lastChild].nextSib = child
	} else {
		p.firstChild = child
	}
	p.lastChild = child
}

func (s *Session) linkLive(parent, child ID) {
	p := s.slots[parent]
	c := s.slots[child]
	c.prevLive = p.lastLive
	if p.lastLive != 0 {
		s.slots[p.lastLive].nextLive = child
	} else {
		p.firstLive = child
	}
	p.lastLive = child
	p.liveN.Add(1)
}

func (s *Session) unlinkLive(child ID) {
	c := s.slots[child]
	parent := c.parent
	if parent == 0 {
		return
	}
	p := s.slots[parent]
	if c.prevLive != 0 {
		s.slots[c.prevLive].nextLive = c.nextLive
	} else {
		p.firstLive = c.nextLive
	}
	if c.nextLive != 0 {
		s.slots[c.nextLive].prevLive = c.prevLive
	} else {
		p.lastLive = c.prevLive
	}
	c.prevLive = 0
	c.nextLive = 0
	p.liveN.Add(-1)
}

func (s *Session) moveLiveFront(child ID) {
	c := s.slots[child]
	parent := c.parent
	if parent == 0 {
		return
	}
	p := s.slots[parent]
	if p.firstLive == child {
		return
	}
	s.unlinkLive(child)
	c.nextLive = p.firstLive
	c.prevLive = 0
	if p.firstLive != 0 {
		s.slots[p.firstLive].prevLive = child
	} else {
		p.lastLive = child
	}
	p.firstLive = child
	p.liveN.Add(1)
}
