package fixed

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/appearance"
)

func init() {
	driver.Register[appearance.Driver](factory{})
}
