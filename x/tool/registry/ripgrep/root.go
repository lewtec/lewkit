package ripgrep

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("ripgrep", "burntsushi/ripgrep", tool.Binary("rg"))
}
