package world

import (
	"reflect"
	"runtime"
	"strings"
)

// Reg is one registered system. After and Before name other systems in
// the same [Set]. The name is the function, so a plugin can refer to a
// system it did not register. Each function is one system in a set. Two
// registrations of the same function literal panic.
type Reg struct {
	sim *Sim
	set Set
	idx int
}

// After runs this system after each prev. A missing name panics when
// the frame is ordered, before the world changes.
func (r *Reg) After(prevs ...System) *Reg {
	if r == nil {
		return nil
	}
	r.sim.fresh = false
	item := &r.sim.systems[r.set][r.idx]
	for _, prev := range prevs {
		if prev == nil {
			continue
		}
		item.after = append(item.after, funcName(prev))
	}
	return r
}

// Before runs this system before each next.
func (r *Reg) Before(nexts ...System) *Reg {
	if r == nil {
		return nil
	}
	r.sim.fresh = false
	item := &r.sim.systems[r.set][r.idx]
	for _, next := range nexts {
		if next == nil {
			continue
		}
		item.before = append(item.before, funcName(next))
	}
	return r
}

type placed struct {
	run     System
	name    string
	after   []string
	before  []string
	seq     int
	lastRun uint64
}

func order(nodes []placed) []int {
	n := len(nodes)
	if n == 0 {
		return nil
	}
	index := make(map[string]int, n)
	for i, node := range nodes {
		index[node.name] = i
	}
	indeg := make([]int, n)
	next := make([][]int, n)
	seen := make(map[[2]int]bool)
	add := func(before, after int, owner string) {
		if before == after {
			panic("world: " + owner + " depends on itself")
		}
		key := [2]int{before, after}
		if seen[key] {
			return
		}
		seen[key] = true
		next[before] = append(next[before], after)
		indeg[after]++
	}
	lookup := func(owner, other string) int {
		j, ok := index[other]
		if !ok {
			panic("world: " + owner + " orders unknown " + other)
		}
		return j
	}
	for i, node := range nodes {
		for _, name := range node.after {
			add(lookup(node.name, name), i, node.name)
		}
		for _, name := range node.before {
			add(i, lookup(node.name, name), node.name)
		}
	}
	ready := make([]int, 0, n)
	for i := range nodes {
		if indeg[i] == 0 {
			ready = append(ready, i)
		}
	}
	out := make([]int, 0, n)
	for len(out) < n {
		if len(ready) == 0 {
			panic("world: system cycle")
		}
		best := 0
		for k := 1; k < len(ready); k++ {
			if nodes[ready[k]].seq < nodes[ready[best]].seq {
				best = k
			}
		}
		i := ready[best]
		ready = append(ready[:best], ready[best+1:]...)
		out = append(out, i)
		for _, j := range next[i] {
			indeg[j]--
			if indeg[j] == 0 {
				ready = append(ready, j)
			}
		}
	}
	return out
}

func funcName(sys System) string {
	if sys == nil {
		return ""
	}
	fn := runtime.FuncForPC(reflect.ValueOf(sys).Pointer())
	if fn == nil {
		return ""
	}
	name := fn.Name()
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}
