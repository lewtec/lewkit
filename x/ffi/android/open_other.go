//go:build !linux

package android

import "context"

func open(context.Context) (*Client, error) {
	return nil, ErrUnavailable
}
