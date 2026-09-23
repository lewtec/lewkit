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

// cpuUnroll expands loop nodes into straight-line ops so the CPU tape
// stays the original interpreter. GLSL still lowers the loop node.
func cpuUnroll(root *node) (*node, []*node, []*buffer, error) {
	if root == nil || !hasLoop(root) {
		return nil, nil, nil, nil
	}
	expanded, err := expandLoops(root, map[*node]*node{})
	if err != nil {
		return nil, nil, nil, err
	}
	order, bufs, err := flatten(expanded)
	if err != nil {
		return nil, nil, nil, err
	}
	return expanded, order, bufs, nil
}

func hasLoop(n *node) bool {
	seen := map[*node]bool{}
	var walk func(*node) bool
	walk = func(n *node) bool {
		if n == nil || seen[n] {
			return false
		}
		seen[n] = true
		if n.kind == kindLoop {
			return true
		}
		for _, source := range n.sources {
			if walk(source) {
				return true
			}
		}
		return false
	}
	return walk(n)
}

func expandLoops(n *node, memo map[*node]*node) (*node, error) {
	if n == nil {
		return nil, nil
	}
	if out, ok := memo[n]; ok {
		return out, nil
	}
	if n.kind == kindLoop {
		out, err := expandOne(n)
		if err != nil {
			return nil, err
		}
		memo[n] = out
		return out, nil
	}
	srcs := make([]*node, len(n.sources))
	same := true
	for i, source := range n.sources {
		next, err := expandLoops(source, memo)
		if err != nil {
			return nil, err
		}
		srcs[i] = next
		if next != source {
			same = false
		}
	}
	if same {
		memo[n] = n
		return n, nil
	}
	clone := *n
	clone.sources = srcs
	memo[n] = &clone
	return &clone, nil
}

func expandOne(loop *node) (*node, error) {
	if len(loop.sources) != 2 || loop.slot < 1 || len(loop.chain) > 0 {
		return nil, ErrOp
	}
	var indexNode, carryNode *node
	findEnds(loop.sources[1], map[*node]bool{}, &indexNode, &carryNode)
	if indexNode == nil || carryNode == nil {
		return nil, ErrOp
	}
	acc, err := expandLoops(loop.sources[0], map[*node]*node{})
	if err != nil {
		return nil, err
	}
	for step := range loop.slot {
		acc, err = substStep(loop.sources[1], indexNode, carryNode, internConst(I32, uint32(int32(step))), acc, map[*node]*node{})
		if err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func findEnds(n *node, seen map[*node]bool, index, carry **node) {
	if n == nil || seen[n] {
		return
	}
	seen[n] = true
	switch n.kind {
	case kindLoopIndex:
		*index = n
	case kindCarry:
		*carry = n
	}
	for _, source := range n.sources {
		findEnds(source, seen, index, carry)
	}
}

func substStep(n, indexNode, carryNode, indexVal, carryVal *node, local map[*node]*node) (*node, error) {
	if n == nil {
		return nil, nil
	}
	if n == indexNode {
		return indexVal, nil
	}
	if n == carryNode {
		return carryVal, nil
	}
	if out, ok := local[n]; ok {
		return out, nil
	}
	if !touchesLoop(n, indexNode, carryNode, map[*node]bool{}) {
		local[n] = n
		return n, nil
	}
	if n.kind == kindGather {
		idx, err := substStep(n.sources[0], indexNode, carryNode, indexVal, carryVal, local)
		if err != nil {
			return nil, err
		}
		idx = simplify(idx)
		if idx == nil || idx.kind != kindConst || idx.dtype != I32 {
			return nil, ErrOp
		}
		loaded, err := scalarLoad(n, int(int32(idx.bits)))
		if err != nil {
			return nil, err
		}
		local[n] = loaded
		return loaded, nil
	}
	srcs := make([]*node, len(n.sources))
	for i, source := range n.sources {
		next, err := substStep(source, indexNode, carryNode, indexVal, carryVal, local)
		if err != nil {
			return nil, err
		}
		srcs[i] = next
	}
	clone := *n
	clone.sources = srcs
	local[n] = &clone
	return &clone, nil
}

func touchesLoop(n, indexNode, carryNode *node, seen map[*node]bool) bool {
	if n == nil || seen[n] {
		return false
	}
	seen[n] = true
	if n == indexNode || n == carryNode {
		return true
	}
	for _, source := range n.sources {
		if touchesLoop(source, indexNode, carryNode, seen) {
			return true
		}
	}
	return false
}

func scalarLoad(gather *node, offset int) (*node, error) {
	if gather == nil || gather.buf == nil || offset < 0 || offset >= gather.buf.cells() {
		return nil, ErrIndex
	}
	base, err := Of(Shape{gather.buf.cells()})
	if err != nil {
		return nil, err
	}
	cell, err := base.Shrink([][2]int{{offset, offset + 1}})
	if err != nil {
		return nil, err
	}
	splat, err := cell.Splat()
	if err != nil {
		return nil, err
	}
	return &node{kind: kindInput, dtype: gather.dtype, buf: gather.buf, tracker: splat}, nil
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
