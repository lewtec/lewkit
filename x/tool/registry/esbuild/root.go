package esbuild

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("esbuild", newEsbuild)
}
