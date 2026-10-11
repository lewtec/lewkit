package clib

import "sync"

// Handler runs one exported C function.
// args holds the integer and pointer registers, in SysV order.
// xmm holds the float registers as raw bits, in SysV order.
// ret is the integer result. retX is the bits of a float result.
type Handler func(args *[6]uintptr, xmm *[8]uint64) (ret uintptr, retX uint64)

var (
	mu       sync.Mutex
	handlers []Handler
)

// Handle records h at id. The generated init calls it once per C function.
// A later call replaces the handler at that id.
func Handle(id int, h Handler) {
	if id < 0 {
		panic("clib: negative handler id")
	}
	mu.Lock()
	defer mu.Unlock()
	for len(handlers) <= id {
		handlers = append(handlers, nil)
	}
	handlers[id] = h
}

func handler(id int32) Handler {
	mu.Lock()
	defer mu.Unlock()
	if id < 0 || int(id) >= len(handlers) {
		return nil
	}
	return handlers[id]
}
