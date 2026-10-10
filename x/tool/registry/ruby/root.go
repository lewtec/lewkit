package ruby

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("ruby", newRuby)
}
