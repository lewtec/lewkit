package d3d12

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/present"
)

var _ driver.DriverFactory[present.Driver] = factory{}

func init() {
	driver.Register[present.Driver](factory{})
}
