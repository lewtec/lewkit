package rtk

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("rtk", "rtk-ai/rtk", tool.Binary("rtk"))
}
