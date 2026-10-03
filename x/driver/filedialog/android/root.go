// Package android shows the Android document picker.
//
// Choose returns content URIs. A folder is one tree URI. [FS] reads those
// URIs. One directory is the root. The bytes stay in the document provider.
// The picker needs the in-process Java VM (an Android build with cgo).
// Call Choose from a goroutine that is not the Android main looper.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

var (
	_ driver.DriverFactory[filedialog.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
