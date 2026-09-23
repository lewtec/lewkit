package ndarray

import "fmt"

// Repeat stacks body count times over init.
// index is the step, 0 .. count-1. carried is the value entering that step.
// The body is one step. The backedge stays on the loop, so arithmetic
// rewrites see a pure function of index and carried.
func Repeat[T Number](init *Tensor[T], count int, body func(index *Tensor[int32], carried *Tensor[T]) *Tensor[T]) (*Tensor[T], error) {
	if init == nil || init.node == nil || body == nil || count < 1 {
		return nil, ErrOp
	}
	if err := init.err(); err != nil {
		return nil, err
	}
	var tracker Tracker
	if shape := init.node.Shape(); shape != nil {
		built, err := Of(shape)
		if err != nil {
			return nil, err
		}
		tracker = built
	}
	index := &node{kind: kindLoopIndex, dtype: I32}
	carry := &node{kind: kindCarry, dtype: init.node.dtype, tracker: tracker}
	out := body(wrap[int32](index), wrap[T](carry))
	if out == nil || out.node == nil {
		return nil, ErrOp
	}
	if out.node.err != nil {
		return nil, out.node.err
	}
	if out.node.dtype != init.node.dtype {
		return nil, ErrType
	}
	loop := &node{
		kind:    kindLoop,
		dtype:   init.node.dtype,
		sources: []*node{init.node, out.node},
		tracker: tracker,
		slot:    count,
	}
	return wrap[T](loop), nil
}

// Gather loads one element. index is an int32 scalar. The result is a splat.
func (t *Tensor[T]) Gather(index *Tensor[int32]) *Tensor[T] {
	if t == nil || index == nil || t.node == nil || index.node == nil {
		return wrap[T](failed(ErrOp))
	}
	if t.node.err != nil {
		return wrap[T](t.node)
	}
	if index.node.err != nil {
		return wrap[T](failed(index.node.err))
	}
	if index.node.dtype != I32 || t.node.kind != kindInput || t.node.buf == nil {
		return wrap[T](failed(ErrOp))
	}
	return wrap[T](&node{
		kind:    kindGather,
		dtype:   t.node.dtype,
		sources: []*node{index.node},
		buf:     t.node.buf,
	})
}

func splitLoop(order []*node) (loop *node, prefix, inside, suffix []*node, err error) {
	for _, n := range order {
		if n.kind != kindLoop {
			continue
		}
		if loop != nil {
			return nil, nil, nil, nil, fmt.Errorf("%w: two loops", ErrOp)
		}
		loop = n
	}
	if loop == nil {
		return nil, order, nil, nil, nil
	}
	if len(loop.sources) != 2 || loop.slot < 1 {
		return nil, nil, nil, nil, ErrOp
	}
	variant := map[*node]bool{}
	markVariant(loop.sources[1], variant)
	seen := false
	for _, n := range order {
		if n == loop {
			seen = true
			continue
		}
		if variant[n] {
			inside = append(inside, n)
			continue
		}
		if seen {
			suffix = append(suffix, n)
		} else {
			prefix = append(prefix, n)
		}
	}
	return loop, prefix, inside, suffix, nil
}

const maxSharedBytes = 8 << 10

func pureSplat(n *node) bool {
	return n != nil && n.kind == kindConst && len(n.chain) == 0 && !n.viewed()
}

// invocationPrivate is true when the value can differ across compute threads.
func invocationPrivate(n *node, memo map[*node]bool) bool {
	if n == nil {
		return false
	}
	if hit, ok := memo[n]; ok {
		return hit
	}
	hit := n.kind == kindCoord || n.kind == kindInput || n.kind == kindCarry
	for _, source := range n.sources {
		if invocationPrivate(source, memo) {
			hit = true
		}
	}
	memo[n] = hit
	return hit
}

// shareGathers lists loop gathers whose address ignores the invocation id.
// The shared table is one slot per iteration, so it has to fit in 8KiB.
func shareGathers(loop *node, inside []*node) []*node {
	if loop == nil || loop.slot < 1 {
		return nil
	}
	memo := map[*node]bool{}
	var gathers []*node
	for _, n := range inside {
		if n.kind != kindGather || invocationPrivate(n, memo) {
			continue
		}
		gathers = append(gathers, n)
	}
	if len(gathers) == 0 || loop.slot*len(gathers)*4 > maxSharedBytes {
		return nil
	}
	return gathers
}

func markVariant(n *node, variant map[*node]bool) bool {
	if n == nil {
		return false
	}
	if hit, ok := variant[n]; ok {
		return hit
	}
	variant[n] = false
	hit := n.kind == kindLoopIndex || n.kind == kindCarry
	for _, source := range n.sources {
		if markVariant(source, variant) {
			hit = true
		}
	}
	variant[n] = hit
	return hit
}
