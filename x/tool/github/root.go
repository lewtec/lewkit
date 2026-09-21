package github

import "github.com/lewtec/lewkit/x/tool"

func init() {
	tool.Register("github", &Backend{})
}
