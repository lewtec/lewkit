package clib

import "sync"

const (
	// KindInt is an integer, pointer, bool, or void result.
	KindInt uint32 = 0
	// KindF32 is a float result. The bits sit in the low half of retX.
	KindF32 uint32 = 1
	// KindF64 is a double result. The bits are retX.
	KindF64 uint32 = 2
)

// Handler runs one exported C function.
// args holds up to eight integer or pointer words.
// xmm holds up to eight float registers as raw bits.
// ret is the integer result. retX is the bits of a float result.
// kind is KindInt, KindF32, or KindF64. The 386 entry uses it to
// load ST(0); the other arches return both registers every time.
type Handler func(args *[8]uint64, xmm *[8]uint64) (ret, retX uint64, kind uint32)

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
