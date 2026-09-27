//go:build android

package jni

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	uithread "github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/thread/std"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[uithread.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "thread_jni" }
func (factory) Name() string { return "Android looper" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java main looper", driver.ErrIncompatible)
	}
	on, err := androidffi.OnLooper()
	if err != nil || !on {
		return fmt.Errorf("%w: no Java main looper", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (uithread.Driver, error) {
	// The looper is already pumping. Jobs still run on this thread when On
	// reports it; callers off the looper use the process queue until a later
	// post lands. The std queue is the fallback inside this process.
	return std.New(), nil
}
