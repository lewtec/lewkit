package fixed

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/colorscheme"
)

func init() {
	driver.Register[colorscheme.Driver](factory{})
}
