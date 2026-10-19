// Package ios writes the iOS pasteboard.
// The app process uses UIPasteboard. A host directory still exchanges request.json.
package ios

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/clipboard"
	"github.com/lewtec/lewkit/x/driver/iosbox"
)

type factory struct{}

func (factory) ID() string   { return "clipboard_ios" }
func (factory) Name() string { return "iOS clipboard" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	return iosbox.Require()
}

func (factory) New(context.Context) (clipboard.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) WriteText(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := iosbox.Call(ctx, iosbox.Request{Op: iosbox.OpClipboard, Text: &text})
	if err != nil {
		return err
	}
	return accepted(raw)
}

func (backend) WriteImage(ctx context.Context, img image.Image) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if img == nil {
		return fmt.Errorf("%w: image", driver.ErrUnavailable)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	raw, err := iosbox.Call(ctx, iosbox.Request{
		Op:  iosbox.OpClipboard,
		PNG: base64.StdEncoding.EncodeToString(buf.Bytes()),
	})
	if err != nil {
		return err
	}
	return accepted(raw)
}

func accepted(raw string) error {
	status, _, err := iosbox.Result(raw)
	if err != nil {
		return err
	}
	if status != askwire.StatusOK {
		return fmt.Errorf("%w: clipboard", driver.ErrUnavailable)
	}
	return nil
}

func init() { driver.Register[clipboard.Driver](factory{}) }

var (
	_ driver.DriverFactory[clipboard.Driver] = factory{}
	_ driver.Weighter                        = factory{}
)
