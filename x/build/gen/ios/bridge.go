package ios

// iosBridgeSource is overlaid into the app main package so c-archive
// exports EletrocromoStart without editing the app tree.
const iosBridgeSource = `//go:build ios

package main

import "C"

import "os"

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
		os.Setenv("ELETROCROMO_CACHE_DIR", C.GoString(cacheDir))
		os.Setenv("LEWKIT_CACHE_DIR", C.GoString(cacheDir))
	}
	if configDir != nil {
		os.Setenv("ELETROCROMO_CONFIG_DIR", C.GoString(configDir))
		os.Setenv("LEWKIT_CONFIG_DIR", C.GoString(configDir))
	}
	os.Setenv("LEWKIT_HOST", "1")
	main()
}
`
