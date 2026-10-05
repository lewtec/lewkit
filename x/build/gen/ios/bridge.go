package ios

// iosBridgeSource is overlaid into the app main package so c-archive
// exports EletrocromoStart without editing the app tree.
const iosBridgeSource = `//go:build ios

package main

import "C"

import (
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/entry"
)

//export EletrocromoStart
func EletrocromoStart(readyFile, dataDir, cacheDir, configDir *C.char) {
	os.Setenv("ELETROCROMO_NO_UI", "1")
	os.Setenv("ELETROCROMO_NO_ENSURE", "1")
	os.Setenv("NO_PROXY", "127.0.0.1,localhost,::1")
	os.Setenv("no_proxy", "127.0.0.1,localhost,::1")
	// c-archive snapshots environ at load. Swift setenv after that is
	// invisible to os.Getenv, so the host must pass dirs here.
	if readyFile != nil {
		os.Setenv("ELETROCROMO_READY_FILE", C.GoString(readyFile))
	}
	if dataDir != nil {
		os.Setenv("ELETROCROMO_DATA_DIR", C.GoString(dataDir))
		os.Setenv("LEWKIT_DATA_DIR", C.GoString(dataDir))
	}
	if cacheDir != nil {
		cache := C.GoString(cacheDir)
		os.Setenv("ELETROCROMO_CACHE_DIR", cache)
		os.Setenv("LEWKIT_CACHE_DIR", cache)
		ask := filepath.Join(cache, "ask")
		_ = os.MkdirAll(ask, 0o700)
		os.Setenv("ELETROCROMO_ASK_DIR", ask)
	}
	if configDir != nil {
		os.Setenv("ELETROCROMO_CONFIG_DIR", C.GoString(configDir))
		os.Setenv("LEWKIT_CONFIG_DIR", C.GoString(configDir))
	}
	main()
}

//export EletrocromoPointer
func EletrocromoPointer(x, y, action C.int) {
	entry.DeliverPointer(int(x), int(y), int(action))
}

//export EletrocromoResize
func EletrocromoResize(width, height C.int) {
	entry.DeliverResize(int(width), int(height))
}

//export EletrocromoDead
func EletrocromoDead(left, top, right, bottom, width, height C.int) {
	entry.DeliverInsets(int(left), int(top), int(right), int(bottom), int(width), int(height))
}

//export EletrocromoSurfaceLost
func EletrocromoSurfaceLost() {
	entry.DeliverSurfaceLost()
}
`
