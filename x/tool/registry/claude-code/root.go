package pkg

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("claude-code", newClaudeCode)
}
