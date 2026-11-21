package applications

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("ruff", "astral-sh/ruff")
}
