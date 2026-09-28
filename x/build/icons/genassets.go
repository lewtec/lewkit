//go:build ignore

// genassets knocks out the default lockup canvas (the only vendored brand asset).
// The square mark is cropped at icon-build time from this lockup.
//
//	go generate ./internal/icons
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/build/icons"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "genassets: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dir := "default"
	path := filepath.Join(dir, "lockup.png")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("run from internal/icons (missing %s): %w", path, err)
	}

	lockup, err := icons.DecodeImage(path)
	if err != nil {
		return err
	}
	lockup = icons.TrimTransparent(icons.KnockoutBackground(lockup), 0.08)
	return icons.WritePNG(path, lockup)
}
