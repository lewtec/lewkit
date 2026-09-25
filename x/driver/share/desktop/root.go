// Package desktop shares text through the clipboard and files through the opener.
package desktop

import (
	"context"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/clipboard"
	"github.com/lewtec/lewkit/x/driver/opener"
	"github.com/lewtec/lewkit/x/driver/share"
)

type factory struct{}

func (factory) ID() string   { return "share_desktop" }
func (factory) Name() string { return "Desktop" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (share.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Out(ctx context.Context, item share.Item) error {
	if len(item.Paths) > 0 {
		return opener.Open(ctx, item.Paths[0])
	}
	text := strings.TrimSpace(item.Text)
	if text == "" {
		text = strings.TrimSpace(item.URL)
	}
	return clipboard.WriteText(ctx, text)
}

func init() { driver.Register[share.Driver](factory{}) }
