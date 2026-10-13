// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

var atExitFuncs []func()

func AtExit(f func()) {
	atExitFuncs = append(atExitFuncs, f)
}

// ExitSignal is the status Exit would have passed to os.Exit.
// The toolchain driver recovers it and keeps the process.
type ExitSignal int

func Exit(code int) {
	for i := len(atExitFuncs) - 1; i >= 0; i-- {
		f := atExitFuncs[i]
		atExitFuncs = atExitFuncs[:i]
		f()
	}
	panic(ExitSignal(code))
}

// To enable tracing support (-t flag), set EnableTrace to true.
const EnableTrace = false
