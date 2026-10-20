package apk

import (
	_ "embed"
	"errors"
)

// hostDex is the Android host (package lewkit) compiled to Dalvik bytecode.
// Resource ids in that dex are the constants in resids.go. It also defines
// every class that bytecode calls outside the Android framework, including
// the swipe-refresh widget and the androidx types that widget calls.
// release build embeds this file. It does not run javac, d8, or aapt.
// lewkit generate androiddex writes it from the Java sources in this package.
//
//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate androiddex
//go:embed classes.dex
var hostDex []byte

// ErrHostDex means the embedded host dex is missing its magic.
var ErrHostDex = errors.New("host dex")
