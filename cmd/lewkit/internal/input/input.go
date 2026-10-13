// Package input reads a CLI byte source from a path or standard input.
package input

import (
	"context"
	"io"
	"os"
)

// Read returns the file at path, or standard input when path is empty or "-".
func Read(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	done := make(chan struct {
		b   []byte
		err error
	}, 1)
	go func() {
		var (
			b   []byte
			err error
		)
		if path != "" && path != "-" {
			b, err = os.ReadFile(path)
		} else {
			b, err = io.ReadAll(os.Stdin)
		}
		done <- struct {
			b   []byte
			err error
		}{b, err}
	}()
	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case got := <-done:
		return got.b, got.err
	}
}
