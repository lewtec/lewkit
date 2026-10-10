package golang

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("golang", newGo)
}
