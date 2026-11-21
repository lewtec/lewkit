package vulkan

import "github.com/lewtec/lewkit/x/driver"

var _ driver.DriverFactory[Device] = factory{}
var _ driver.Offerer[Device] = factory{}

func init() {
	driver.Register[Device](factory{})
}
