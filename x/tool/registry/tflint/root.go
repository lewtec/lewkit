package pkg

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("tflint", "terraform-linters/tflint", tool.Binary("tflint"))
}
