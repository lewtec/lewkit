//go:build android && cgo

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[battery.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "battery_android" }
func (factory) Name() string { return "Android battery" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (battery.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) BatteryStatus(context.Context) (battery.Status, error) {
	text, err := androidffi.StaticString("batteryStatus")
	if err != nil {
		return battery.Unknown, err
	}
	return statusFromHost(text)
}
