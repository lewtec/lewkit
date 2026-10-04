// Package android opens a URL or file through an Android view intent.
//
// A leading slash is a file under the app file-provider roots.
// Any other target needs a URI scheme, such as https.
package android

import (
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

var _ driver.DriverFactory[opener.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() { driver.Register[opener.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "opener_android" }
func (factory) Name() string { return "Android open" }
func (factory) Weight() int  { return 80 }
