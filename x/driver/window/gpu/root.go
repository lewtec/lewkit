package gpu

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func init() {
	driver.Register[window.Driver](factory{})
}
