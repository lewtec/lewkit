//go:build linux

package android

import (
	"context"
	"errors"
	"fmt"

	gbinder "github.com/AndroidGoLab/binder/binder"
	"github.com/AndroidGoLab/binder/binder/versionaware"
	"github.com/AndroidGoLab/binder/kernelbinder"
	"github.com/AndroidGoLab/binder/servicemanager"
	selinux "github.com/opencontainers/selinux/go-selinux"
)

func init() {
	// A binder status error asks for the process SELinux label. That lookup
	// probes open_tree(2). Android's app seccomp kills that syscall with
	// SIGSYS instead of returning ENOSYS.
	selinux.SetDisabled()
}

func open(ctx context.Context) (*Client, error) {
	drv, err := kernelbinder.Open(ctx, gbinder.WithMapSize(128*1024))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	transport, err := versionaware.NewTransport(ctx, drv, 0)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%w: %w", ErrUnavailable, err), drv.Close(ctx))
	}
	return &Client{call: &raw{sm: servicemanager.New(transport)}}, nil
}
