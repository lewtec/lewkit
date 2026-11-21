package native

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/httpclient"
)

func init() {
	driver.Register[httpclient.Driver](factory{})
}
