// Command toolchain builds a Go shared library and runs generated source.
//
//	go run ./examples/toolchain share
//	go run ./examples/toolchain jit
//
// share renders a main package with jennifer, compiles it into a shared
// object, loads it with purego, and calls HostPing. jit renders a main
// package with jennifer, compiles it in this process, and calls Value.
package main

import (
	"context"
	"fmt"
	"os"

	gotool "github.com/lewtec/lewkit/x/toolchain/go"
)

func main() {
	if handled, err := childOrShare(); handled {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) != 2 {
		usage()
		os.Exit(2)
	}
	ctx := context.Background()
	var (
		out string
		err error
	)
	switch os.Args[1] {
	case "share":
		out, err = share(ctx)
	case "jit":
		out, err = jit(ctx)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(out)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: toolchain share|jit")
}

// childOrShare reports whether this process is a compiler or linker child,
// or the process that calls the shared library. The caller exits.
func childOrShare() (bool, error) {
	if handled, err := gotool.Child(); handled {
		return true, err
	}
	if so := os.Getenv("LEWKIT_SHARE_SO"); so != "" {
		return true, callShared(so)
	}
	return false, nil
}
