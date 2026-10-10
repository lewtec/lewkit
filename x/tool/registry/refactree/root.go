package refactree

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("refactree", "lucasew/refactree", tool.Binary("rft"))
}
