// Package prelude registers the standard tool backends and curated short names.
//
// Blank-import this package before tool.Ensure so github, mise, and registry specs resolve.
package prelude

import (
	_ "github.com/lewtec/lewkit/x/tool/github"
	_ "github.com/lewtec/lewkit/x/tool/mise"
	_ "github.com/lewtec/lewkit/x/tool/registry"
	_ "github.com/lewtec/lewkit/x/tool/registry/applications"
)
