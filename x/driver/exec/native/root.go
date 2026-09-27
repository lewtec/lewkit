// Package native runs programs with os/exec.
package native

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type factory struct{}

func (factory) ID() string   { return "exec_native" }
func (factory) Name() string { return "native" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (execdriver.Driver, error) { return backend{}, nil }

func init() {
	driver.Register[execdriver.Driver](factory{})
}
