// Package table keeps one program per kernel for a GPU evaluator.
package table

import (
	"sync"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Program is one compiled kernel the table can close.
type Program interface {
	comparable
	Close() error
}

// Table is the kernel-to-program map and the lock Eval holds.
// Eval is the mutex a session takes for the duration of one evaluation.
type Table[P Program] struct {
	mu    sync.Mutex
	Eval  sync.Mutex
	items map[*ndarray.Kernel]P
}

// With runs fn while holding the table lock.
func (table *Table[P]) With(fn func()) {
	table.mu.Lock()
	defer table.mu.Unlock()
	fn()
}

// Open returns the program already stored for kernel when same accepts it.
// Otherwise it stores the program create returns. attach runs on that new
// program before it is stored. fresh is true when the returned program is
// the one create just made. A close error from the program that lost the
// race is returned, and the winner stays in the table.
func (table *Table[P]) Open(kernel *ndarray.Kernel, same func(P) bool, create func() (P, error), attach func(P)) (kept P, fresh bool, err error) {
	if found, ok := table.lookup(kernel, same); ok {
		return found, false, nil
	}
	created, err := create()
	if err != nil {
		var zero P
		return zero, false, err
	}
	if attach != nil {
		attach(created)
	}
	kept, err = table.install(kernel, created, same)
	if err != nil {
		var zero P
		return zero, false, err
	}
	return kept, kept == created, nil
}

// Forget removes kernel when it still points at program.
// The func is what a session calls from Close.
func (table *Table[P]) Forget(kernel *ndarray.Kernel, program P) func() {
	return func() {
		table.mu.Lock()
		defer table.mu.Unlock()
		if table.items != nil && table.items[kernel] == program {
			delete(table.items, kernel)
		}
	}
}

// Take detaches every program and runs fn while holding the lock.
// fn clears evaluator fields that share this lock. The returned slice
// is what the caller closes after the lock is released.
func (table *Table[P]) Take(fn func()) []P {
	table.mu.Lock()
	items := table.items
	table.items = nil
	if fn != nil {
		fn()
	}
	table.mu.Unlock()
	out := make([]P, 0, len(items))
	for _, program := range items {
		out = append(out, program)
	}
	return out
}

func (table *Table[P]) lookup(kernel *ndarray.Kernel, same func(P) bool) (P, bool) {
	table.mu.Lock()
	defer table.mu.Unlock()
	if table.items == nil {
		table.items = make(map[*ndarray.Kernel]P)
	}
	program := table.items[kernel]
	var zero P
	if program != zero && same(program) {
		return program, true
	}
	return zero, false
}

func (table *Table[P]) install(kernel *ndarray.Kernel, fresh P, same func(P) bool) (P, error) {
	table.mu.Lock()
	existing := table.items[kernel]
	var zero P
	if existing != zero && same(existing) {
		table.mu.Unlock()
		if err := fresh.Close(); err != nil {
			return zero, err
		}
		return existing, nil
	}
	old := table.items[kernel]
	table.items[kernel] = fresh
	table.mu.Unlock()
	if old != zero {
		if err := old.Close(); err != nil {
			return zero, err
		}
	}
	return fresh, nil
}
