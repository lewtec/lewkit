// Package ios shows the iOS document picker through the packaged host.
// Choose copies a selection into the cache inbox and returns that path.
// Save returns a new file in the inbox after the export sheet.
package ios

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/iosbox"
)

type factory struct{}

func (factory) ID() string   { return "filedialog_ios" }
func (factory) Name() string { return "iOS document picker" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	if !iosbox.Available() {
		return fmt.Errorf("%w: no ios host", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (filedialog.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Choose(ctx context.Context, req filedialog.Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", filedialog.ErrRequest, err)
	}
	raw, err := iosbox.Call(ctx, iosbox.Request{
		Op:        iosbox.OpPick,
		Title:     req.Title,
		Save:      req.Save,
		Folder:    req.Folder,
		Multiple:  req.Multiple,
		Name:      req.Name,
		Directory: req.Directory,
		Exts:      filedialog.Extensions(req.Filters),
	})
	if err != nil {
		return nil, err
	}
	return pathsFrom(raw)
}

func pathsFrom(raw string) ([]string, error) {
	status, payload, err := iosbox.Result(raw)
	if err != nil {
		return nil, err
	}
	if status == askwire.StatusCanceled {
		return nil, filedialog.ErrCanceled
	}
	payload = strings.TrimSpace(payload)
	if payload == "" || payload == "[]" {
		return nil, filedialog.ErrCanceled
	}
	var paths []string
	if err := json.Unmarshal([]byte(payload), &paths); err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, filedialog.ErrCanceled
	}
	return paths, nil
}

func init() { driver.Register[filedialog.Driver](factory{}) }

var (
	_ driver.DriverFactory[filedialog.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
