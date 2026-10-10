package registry

import "github.com/lewtec/lewkit/x/tool"

func init() {
	tool.Register("registry", &catalog{})
}
