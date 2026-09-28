// Package bundle is the on-disk root of the binary's stamped application id.
//
//	root, err := bundle.Resolve(ctx)
//
// The id comes from [release.AppID]. Data, cache, and config come from
// [dirs.Resolve]. Profile is Data/webview, the directory a web view uses
// for cookies and storage.
package bundle

import (
	"context"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/release"
)

// Root is one application's private tree.
type Root struct {
	ID      string
	Data    string
	Cache   string
	Config  string
	Profile string
}

// Driver resolves a tree for one reverse-domain id.
type Driver interface {
	Resolve(ctx context.Context, appID string) (Root, error)
}

// Resolve returns the tree for the id stamped into this binary.
func Resolve(ctx context.Context) (Root, error) {
	id, err := release.AppID()
	if err != nil {
		return Root{}, err
	}
	return ResolveID(ctx, id)
}

// ResolveID returns the tree for one reverse-domain id.
func ResolveID(ctx context.Context, id string) (Root, error) {
	if err := release.ValidateAppID(id); err != nil {
		return Root{}, err
	}
	return driver.WithResult(ctx, func(d Driver) (Root, error) {
		return d.Resolve(ctx, id)
	})
}

// SharePath is the JSONL drop the packaged host tails.
func SharePath(root Root) string {
	return filepath.Join(root.Cache, "share.jsonl")
}
