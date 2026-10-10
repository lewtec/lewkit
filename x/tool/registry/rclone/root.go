package rclone

import (
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterGitHub("rclone", "rclone/rclone", tool.Binary("rclone"))
}
