//go:build darwin

package native

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var (
	selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")
	selUTF8String     = objc.RegisterName("UTF8String")
)

// NSString returns text as an NSString.
// The NUL-terminated bytes stay live across the Objective-C call.
func NSString(text string) objc.ID {
	raw := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8, unsafe.Pointer(&raw[0]))
}

// UTF8String copies the NUL-terminated UTF-8 bytes of an Objective-C string.
// A zero id, a missing pointer, or no NUL within 1<<20 bytes is empty.
// GoString returns the bytes it read when the NUL is missing.
func UTF8String(id objc.ID) string {
	if id == 0 {
		return ""
	}
	pointer := id.Send(selUTF8String)
	if pointer == 0 {
		return ""
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(pointer)), 1<<20)
	for i, b := range raw {
		if b == 0 {
			return string(raw[:i])
		}
	}
	return ""
}
