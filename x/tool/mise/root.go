package mise

import "github.com/lewtec/lewkit/x/tool"

func init() {
	tool.Register("mise", &Backend{})
}
