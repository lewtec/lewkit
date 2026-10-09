//go:build !linux

package native

import "context"

func hostLibDirs() []string { return nil }

// Prepare is the non-Linux form. There is no toolkit lookup.
func Prepare(ctx context.Context) error {
	return ctx.Err()
}
