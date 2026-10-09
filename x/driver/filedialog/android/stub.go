//go:build !android

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

func init() { driver.Register[filedialog.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "filedialog_android" }
func (factory) Name() string { return "Android file chooser" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}

func (factory) New(context.Context) (filedialog.Driver, error) {
	return nil, fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
}
