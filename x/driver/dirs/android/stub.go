//go:build !android || !cgo

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/dirs"
)

func init() { driver.Register[dirs.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "dirs_android" }
func (factory) Name() string { return "Android directories" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

func (factory) New(context.Context) (dirs.Driver, error) {
	return nil, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
