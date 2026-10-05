package common

import (
	"context"
	"strings"

	"github.com/lewtec/lewkit/x/build/version"
)

// StampPackagingVersion resolves app-tree version identity for a host Build.
// Empty explicitName / non-positive explicitCode fall back to git/ldflags in goMain.
// ctx is the caller's context. Git reads stop when it ends.
func StampPackagingVersion(ctx context.Context, goMain, explicitName string, explicitCode int) (vi version.Info, name string, code int) {
	vi = version.ResolveDir(ctx, goMain)
	name = explicitName
	code = explicitCode
	if strings.TrimSpace(name) == "" {
		name = vi.AndroidName()
	}
	if code <= 0 {
		code = version.AndroidCodeFrom(vi.Version, version.GitCommitCount(ctx, goMain))
	}
	return vi, name, code
}
