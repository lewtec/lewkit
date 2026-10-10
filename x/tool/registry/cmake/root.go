package cmake

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("cmake", newCMake)
}
