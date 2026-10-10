package pkg

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("resvg", "linebender/resvg", tool.Binary("resvg"))
}
