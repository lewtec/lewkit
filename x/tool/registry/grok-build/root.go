package pkg

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("grok-build", newGrokBuild)
}
