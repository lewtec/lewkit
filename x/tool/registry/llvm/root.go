package llvm

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("llvm", newLLVM)
}
