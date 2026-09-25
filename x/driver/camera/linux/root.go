// Package linux captures V4L2 cameras with ffmpeg.
package linux

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/camera"
)

type factory struct{}

func (factory) ID() string   { return "camera_v4l" }
func (factory) Name() string { return "V4L2 + ffmpeg" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: linux is required", driver.ErrIncompatible)
	}
	return requireBinary("ffmpeg")
}

func (factory) New(context.Context) (camera.Driver, error) { return backend{}, nil }

var _ driver.DriverFactory[camera.Driver] = factory{}
var _ driver.Weighter = factory{}

func init() {
	driver.Register[camera.Driver](factory{})
}
