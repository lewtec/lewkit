package common

import (
	"strings"

	"github.com/lewtec/lewkit/x/build/version"
)

// StampPackagingVersion resolves app-tree version identity for a host Build.
// Empty explicitName / non-positive explicitCode fall back to git/ldflags in goMain.
func StampPackagingVersion(goMain, explicitName string, explicitCode int) (vi version.Info, name string, code int) {
	vi = version.ResolveDir(goMain)
	name = explicitName
	code = explicitCode
	if strings.TrimSpace(name) == "" {
		name = vi.AndroidName()
	}
	if code <= 0 {
		code = version.AndroidCodeFrom(vi.Version, version.GitCommitCount(goMain))
	}
	return vi, name, code
}
