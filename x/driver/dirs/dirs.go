// Package dirs resolves per-app storage locations.
//
//	tree, err := dirs.Resolve(ctx, "br.tec.lew.myapp")
//
// Data, Cache, and Config sit under the OS homes, joined with the app id.
// Inbox is Cache/inbox. LEWKIT_DATA_DIR, LEWKIT_CACHE_DIR, and
// LEWKIT_CONFIG_DIR replace those three paths when set.
package dirs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

// ErrInvalidAppID means the app id is empty or contains a path element.
var ErrInvalidAppID = errors.New("invalid app id")

// Dirs is the app-private tree. Inbox is a subdirectory of Cache.
type Dirs struct {
	Data   string
	Cache  string
	Config string
	Inbox  string
}

// Driver resolves directories for one app id.
type Driver interface {
	Resolve(ctx context.Context, appID string) (Dirs, error)
}

// Resolve returns locations for appID.
func Resolve(ctx context.Context, appID string) (Dirs, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" || strings.Contains(appID, "..") || strings.ContainsAny(appID, `/\`) {
		return Dirs{}, fmt.Errorf("%w: %q", ErrInvalidAppID, appID)
	}
	return driver.WithResult(ctx, func(d Driver) (Dirs, error) {
		return d.Resolve(ctx, appID)
	})
}
