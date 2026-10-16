//go:build !android

package android

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
)

func open(context.Context) (backend, error) {
	return backend{}, driver.JavaVM()
}

type backend struct{}

func (backend) Choose(context.Context, launcher.ChooseOptions) (*launcher.Item, error) {
	return nil, driver.JavaVM()
}

func (backend) Prompt(context.Context, string) (string, error) {
	return "", driver.JavaVM()
}

func (backend) Confirm(context.Context, string) (bool, error) {
	return false, driver.JavaVM()
}
