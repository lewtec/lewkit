// Package osascript posts a notification with the osascript command.
package osascript

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type factory struct{}

func (factory) ID() string   { return "notification_osascript" }
func (factory) Name() string { return "Notification Center" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	return execdriver.RequireBinary(ctx, "osascript")
}

func (factory) New(context.Context) (notification.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Notify(ctx context.Context, n notification.Notification) error {
	cmd := execdriver.MustCommand("osascript", "-e", script(n))
	if err := execdriver.Run(ctx, cmd); err != nil {
		return fmt.Errorf("osascript: %w", err)
	}
	return nil
}

func script(n notification.Notification) string {
	title := n.Title
	if strings.TrimSpace(title) == "" {
		title = "Notification"
	}
	text := "display notification " + apple(n.Message) + " with title " + apple(title)
	if n.Urgency == "critical" {
		text += ` sound name "Basso"`
	}
	return text
}

func apple(text string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range text {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n', '\r':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func init() { driver.Register[notification.Driver](factory{}) }

var (
	_ driver.DriverFactory[notification.Driver] = factory{}
	_ driver.Weighter                           = factory{}
)
