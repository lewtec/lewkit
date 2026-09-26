// Package host appends share.Out to the bundle share drop.
// The packaged native host tails that file and opens the system share sheet.
// Compatible on Android, and when LEWKIT_HOST is 1, true, or yes.
package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/bundle"
	_ "github.com/lewtec/lewkit/x/driver/bundle/prelude"
	_ "github.com/lewtec/lewkit/x/driver/dirs/prelude"
	"github.com/lewtec/lewkit/x/driver/share"
)

type factory struct{}

func (factory) ID() string   { return "share_host" }
func (factory) Name() string { return "Host share drop" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS == "android" || envTruthy("LEWKIT_HOST") {
		return nil
	}
	return fmt.Errorf("%w: not a packaged host", driver.ErrIncompatible)
}

func (factory) New(context.Context) (share.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Out(ctx context.Context, item share.Item) error {
	root, err := bundle.Resolve(ctx)
	if err != nil {
		return err
	}
	line, err := json.Marshal(drop{
		Title: item.Title,
		Text:  item.Text,
		URL:   item.URL,
		Paths: item.Paths,
	})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(bundle.SharePath(root), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(line, '\n'))
	return err
}

type drop struct {
	Title string   `json:"title,omitempty"`
	Text  string   `json:"text,omitempty"`
	URL   string   `json:"url,omitempty"`
	Paths []string `json:"paths,omitempty"`
}

func envTruthy(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func init() { driver.Register[share.Driver](factory{}) }
