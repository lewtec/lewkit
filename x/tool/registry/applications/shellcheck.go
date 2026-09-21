package applications

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("shellcheck", "koalaman/shellcheck")
}
