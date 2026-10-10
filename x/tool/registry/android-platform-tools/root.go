package androidplatformtools

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	registry.RegisterTool("android-platform-tools", newAndroidPlatformTools)
}
