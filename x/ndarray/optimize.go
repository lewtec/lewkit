package ndarray

// optimize rewrites the node graph before any backend lowers it.
// Rules run bottom-up, in slice order. The first match wins.
// CPU and GLSL both consume the rewritten graph.

const optRewriteLimit = 4096

type optRule func(*node) *node

// optRules is the rewrite order. The first match wins.
var optRules = []optRule{
	foldConstOp,
	matchIdentity,
	commuteIntConst,
	assocIntConst,
	matchNegNeg,
	matchWhereSame,
	matchIntSrc,
}

type rewriter struct {
	memo map[*node]*node
	cse  map[string]*node
	left int
}

func optimize(n *node) *node {
	r := &rewriter{
		memo: map[*node]*node{},
		cse:  map[string]*node{},
		left: optRewriteLimit,
	}
	return r.node(n)
}

func (r *rewriter) node(n *node) *node {
	if n == nil {
		return nil
	}
	if m := r.memo[n]; m != nil {
		return m
	}
	if n.kind != kindOp {
		r.memo[n] = n
		return n
	}
	srcs := make([]*node, len(n.sources))
	same := true
	for i, s := range n.sources {
		srcs[i] = r.node(s)
		if srcs[i] != s {
			same = false
		}
	}
	out := n
	if !same {
		clone := *n
		clone.sources = srcs
		out = &clone
	}
	for r.left > 0 {
		next := applyOpt(out)
		if next == nil || next == out {
			break
		}
		r.left--
		out = r.node(next)
		if out == nil || out.kind != kindOp {
			r.memo[n] = out
			return out
		}
	}
	if out.kind != kindOp {
		r.memo[n] = out
		return out
	}
	key := cseKey(out)
	if hit := r.cse[key]; hit != nil {
		r.memo[n] = hit
		return hit
	}
	r.cse[key] = out
	r.memo[n] = out
	return out
}

func applyOpt(n *node) *node {
	for _, rule := range optRules {
		if next := rule(n); next != nil && next != n {
			return next
		}
	}
	return nil
}

func matchIdentity(n *node) *node {
	out := identityOp(n)
	if out == n {
		return nil
	}
	return out
}

func plainOp(n *node) bool {
	return n != nil && n.kind == kindOp && !n.viewed() && len(n.chain) == 0
}

func splatConst(n *node) bool {
	return n != nil && n.kind == kindConst && len(n.tracker.views) == 0 && len(n.chain) == 0
}

func intArith(n *node) bool {
	return plainOp(n) && n.dtype == I32 && (n.op == ADD || n.op == MUL)
}

// commuteIntConst parks a splat on the right so later rules see one shape.
func commuteIntConst(n *node) *node {
	if !intArith(n) || len(n.sources) != 2 {
		return nil
	}
	if !splatConst(n.sources[0]) || splatConst(n.sources[1]) {
		return nil
	}
	return opNode(n.op, n.dtype, n.sources[1], n.sources[0])
}

// assocIntConst folds (x ⊕ c1) ⊕ c2 into x ⊕ (c1 ⊕ c2) for int add and mul.
func assocIntConst(n *node) *node {
	if !intArith(n) || len(n.sources) != 2 || !splatConst(n.sources[1]) {
		return nil
	}
	left := n.sources[0]
	if !intArith(left) || left.op != n.op || len(left.sources) != 2 || !splatConst(left.sources[1]) {
		return nil
	}
	bits := instruction{alu: n.op, dtype: I32, inType: I32, a: 0, b: 1}.evalALU([]uint32{left.sources[1].bits, n.sources[1].bits})
	return opNode(n.op, I32, left.sources[0], internConst(I32, bits))
}

func matchNegNeg(n *node) *node {
	if !plainOp(n) || n.op != NEG || (n.dtype != I32 && n.dtype != F32) {
		return nil
	}
	inner := n.sources[0]
	if !plainOp(inner) || inner.op != NEG || inner.dtype != n.dtype {
		return nil
	}
	return inner.sources[0]
}

func matchWhereSame(n *node) *node {
	if !plainOp(n) || n.op != WHERE || len(n.sources) != 3 {
		return nil
	}
	if n.sources[1] != n.sources[2] {
		return nil
	}
	return n.sources[1]
}

func matchIntSrc(n *node) *node {
	if !plainOp(n) || n.dtype != I32 || len(n.sources) != 2 {
		return nil
	}
	left, right := n.sources[0], n.sources[1]
	switch n.op {
	case SHL, SHR, IDIV:
		if (n.op == IDIV && isOneConst(right)) || (n.op != IDIV && isZeroConst(right)) {
			return left
		}
	case OR, XOR:
		if isZeroConst(right) {
			return left
		}
		if isZeroConst(left) {
			return right
		}
	case AND:
		if splatConst(right) && right.bits == ^uint32(0) {
			return left
		}
		if splatConst(left) && left.bits == ^uint32(0) {
			return right
		}
	}
	return nil
}
