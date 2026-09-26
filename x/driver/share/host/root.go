// Package host appends share.Out to the bundle share drop.
// The packaged native host tails that file and opens the system share sheet.
// The factory is compatible only in the android and ios files.
package host

import (
	"context"
	"encoding/json"
	"os"

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

func init() { driver.Register[share.Driver](factory{}) }
