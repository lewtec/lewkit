package vulkanwindow

import "github.com/lewtec/lewkit/x/driver"

func init() {
	driver.Register[Driver](factory{})
}
