package ndeval

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ndarray"
)

func init() {
	driver.Register[ndarray.Evaluator](cpuFactory{})
}

type cpuFactory struct{}

func (cpuFactory) ID() string                               { return "ndeval_cpu" }
func (cpuFactory) Name() string                             { return "CPU" }
func (cpuFactory) Weight() int                              { return 0 }
func (cpuFactory) CheckCompatibility(context.Context) error { return nil }
func (cpuFactory) New(context.Context) (ndarray.Evaluator, error) {
	return ndarray.CPU, nil
}
