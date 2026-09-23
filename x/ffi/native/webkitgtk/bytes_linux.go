//go:build linux

package webkitgtk

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

type pinnedBytes struct {
	bytes  []byte
	pinner runtime.Pinner
}

var pinnedBlobs sync.Map

// destroyPinnedBytes is the GDestroyNotify for memory streams.
var destroyPinnedBytes = purego.NewCallback(func(data uintptr) {
	value, ok := pinnedBlobs.LoadAndDelete(data)
	if !ok {
		return
	}
	value.(*pinnedBytes).pinner.Unpin()
})

func pinBytes(body []byte) uintptr {
	copied := append([]byte(nil), body...)
	blob := &pinnedBytes{bytes: copied}
	blob.pinner.Pin(&blob.bytes[0])
	pointer := uintptr(unsafe.Pointer(&blob.bytes[0]))
	pinnedBlobs.Store(pointer, blob)
	return pointer
}

func cStringBytes(text string) []byte {
	return append([]byte(text), 0)
}

func cStringPointer(text []byte) *byte {
	if len(text) == 0 {
		return nil
	}
	return &text[0]
}
