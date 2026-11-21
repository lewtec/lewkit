package applications

import (
	"github.com/lewtec/lewkit/x/tool/registry"
)

func init() {
	// Official shfmt lives under mvdan/sh (patrickvane/shfmt is a stale fork
	// whose release assets lack linux/arm64).
	registry.RegisterGitHub("shfmt", "mvdan/sh")
}
