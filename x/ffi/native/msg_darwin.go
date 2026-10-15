//go:build darwin

package native

import "sync"

var (
	msgOnce sync.Once
	msgAddr uintptr
)

// MsgSend is the address of objc_msgSend.
// A missing libobjc is 0.
func MsgSend() uintptr {
	msgOnce.Do(func() {
		lib, err := Open("/usr/lib/libobjc.A.dylib", Lazy)
		if err != nil {
			return
		}
		addr, err := Symbol(lib, "objc_msgSend")
		if err != nil {
			return
		}
		msgAddr = addr
	})
	return msgAddr
}
