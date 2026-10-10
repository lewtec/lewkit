package fzf

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("fzf", "junegunn/fzf", tool.Binary("fzf"))
}
