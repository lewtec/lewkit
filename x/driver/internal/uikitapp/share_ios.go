//go:build ios

package uikitapp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
)

var selInitActivity = objc.RegisterName("initWithActivityItems:applicationActivities:")

// Share presents the share sheet and returns once it is on screen.
func Share(ctx context.Context, title, text, link string, paths []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var failed error
	err := onUI(func() {
		withPool(func() {
			failed = showShare(title, text, link, paths)
		})
	})
	if err != nil {
		return err
	}
	return failed
}

func showShare(title, text, link string, paths []string) error {
	if err := ensure(); err != nil {
		return err
	}
	if text == "" {
		text = strings.TrimSpace(title)
	}
	items := cls("NSMutableArray").Send(selArray)
	if text != "" {
		items.Send(selAdd, ns(text))
	}
	if link != "" {
		if url := urlFrom(link); url != 0 {
			items.Send(selAdd, url)
		}
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if url := cls("NSURL").Send(selFileURL, ns(path)); url != 0 {
			items.Send(selAdd, url)
		}
	}
	if items == 0 || int(items.Send(selCount)) == 0 {
		return fmt.Errorf("%w: share", driver.ErrUnavailable)
	}
	ctrl := cls("UIActivityViewController").Send(selAlloc).Send(selInitActivity, items, objc.ID(0))
	if ctrl == 0 {
		return fmt.Errorf("%w: share", driver.ErrUnavailable)
	}
	presenter := topController()
	if presenter == 0 {
		ctrl.Send(selRelease)
		return fmt.Errorf("%w: no window", driver.ErrUnavailable)
	}
	present(presenter, ctrl)
	ctrl.Send(selRelease)
	return nil
}
