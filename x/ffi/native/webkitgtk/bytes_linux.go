//go:build linux && !android

package webkitgtk

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// pinnedBytes is the copy of a scheme body whose address a GMemoryInputStream holds.
// GDestroyNotify receives only that address, so the map is how Drop finds the pin.
// Drop runs from the notify, and from FinishRequest when the stream is not created.
type pinnedBytes struct {
	mu     sync.Mutex
	bytes  []byte
	pinner runtime.Pinner
	ptr    uintptr
}

var pinnedBlobs sync.Map

// destroyPinnedBytes is the GDestroyNotify for a memory stream.
var destroyPinnedBytes = purego.NewCallback(func(data uintptr) {
	dropPinned(data)
})

func dropPinned(data uintptr) {
	value, ok := pinnedBlobs.Load(data)
	if !ok {
		return
	}
	value.(*pinnedBytes).Drop()
}

// pinBytes copies body and pins the copy. An empty body has nothing to pin.
func pinBytes(body []byte) *pinnedBytes {
	if len(body) == 0 {
		return nil
	}
	copied := append([]byte(nil), body...)
	blob := &pinnedBytes{bytes: copied}
	blob.pinner.Pin(&blob.bytes[0])
	blob.ptr = uintptr(unsafe.Pointer(&blob.bytes[0]))
	pinnedBlobs.Store(blob.ptr, blob)
	return blob
}

func (blob *pinnedBytes) pointer() uintptr {
	if blob == nil {
		return 0
	}
	return blob.ptr
}

// Drop unpins the copy. A second call does nothing.
func (blob *pinnedBytes) Drop() {
	if blob == nil {
		return
	}
	blob.mu.Lock()
	defer blob.mu.Unlock()
	if blob.ptr == 0 {
		return
	}
	ptr := blob.ptr
	blob.ptr = 0
	pinnedBlobs.Delete(ptr)
	blob.pinner.Unpin()
}
