package native

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/fetchurl"
)

func init() {
	driver.Register[fetchurl.Driver](factory{})
}
