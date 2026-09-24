package fixed

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
)

func init() {
	driver.Register[daynight.Driver](factory{})
}
