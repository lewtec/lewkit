package win32

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/appearance"
)

type factory struct{}

func (factory) ID() string   { return "appearance_win32" }
func (factory) Name() string { return "Personalize" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error { return available(ctx) }

func (factory) New(ctx context.Context) (appearance.Driver, error) { return open(ctx) }

func init() {
	driver.Register[appearance.Driver](factory{})
}
