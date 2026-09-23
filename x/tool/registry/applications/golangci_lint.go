package applications

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("golangci-lint", "golangci/golangci-lint")
}
