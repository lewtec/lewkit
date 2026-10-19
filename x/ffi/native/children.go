package native

import (
	"reflect"
	"sync"
)

// Child is one native subobject. Drop frees its handles.
// A second call does nothing.
type Child interface {
	Drop()
}

// Children is the set of subobjects an owner drops before itself.
// The key is the Go object address. Add, Forget, and Drop may run
// together. Drop calls each child without holding the set lock, so
// Drop on the child may Forget that child.
type Children struct {
	mu sync.Mutex
	m  map[uintptr]Child
}

// Add records obj. A nil obj is ignored. Adding the same obj twice keeps one entry.
func (c *Children) Add(obj Child) {
	if c == nil || obj == nil {
		return
	}
	key := addrOf(obj)
	if key == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[uintptr]Child)
	}
	c.m[key] = obj
}

// Forget removes obj. The caller still drops it.
func (c *Children) Forget(obj Child) {
	if c == nil || obj == nil {
		return
	}
	key := addrOf(obj)
	if key == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}

// Drop frees every child, then forgets them.
func (c *Children) Drop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	items := c.m
	c.m = nil
	c.mu.Unlock()
	for _, obj := range items {
		obj.Drop()
	}
}

func addrOf(obj any) uintptr {
	if obj == nil {
		return 0
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return 0
	}
	return uintptr(v.Pointer())
}
