package opencode

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("opencode", "anomalyco/opencode", tool.Binary("opencode"))
}
