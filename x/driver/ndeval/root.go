package ndeval

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ndarray"
)

var _ driver.DriverFactory[ndarray.Evaluator] = cpuFactory{}
var _ driver.DriverFactory[ndarray.Evaluator] = gpuFactory{}

func init() {
	driver.Register[ndarray.Evaluator](cpuFactory{})
	driver.Register[ndarray.Evaluator](gpuFactory{})
}
