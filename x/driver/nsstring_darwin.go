//go:build darwin

package driver

import (
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

var selStringWithUTF8 = objc.RegisterName("stringWithUTF8String:")

// NSString returns text as an NSString.
func NSString(text string) objc.ID {
	raw := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selStringWithUTF8, unsafe.Pointer(&raw[0]))
}
