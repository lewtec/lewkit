//go:build !linux || !amd64

package clib

import "context"

// Build links a shared library on linux/amd64.
func Build(context.Context, string, string) error {
	return ErrHost
}
