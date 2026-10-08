package vulkan

import (
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/prelude"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/native/vulkan"
)

func init() {
	ffivulkan.SetUIThread(processThread{})
}

type processThread struct{}

func (processThread) Do(fn func())      { thread.Do(fn) }
func (processThread) Bound() bool       { return thread.Bound() }
func (processThread) ProcessMain() bool { return thread.On() }
func (processThread) OnIdle(fn func())  { thread.OnIdle(fn) }
