// Package tray opens a host status item and its menu.
//
//	item, err := tray.Open(ctx, tray.Config{
//		Title: "lewkit",
//		Icon:  tray.Icon{Image: img}, // or PNG, JPEG, ICO, or ICNS bytes
//		Menu:  []tray.Item{{Label: "Quit", OnClick: cancel}},
//	})
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or one implementation
// (dbus, win32, cocoa).
//
// Linux speaks org.kde.StatusNotifierItem and com.canonical.dbusmenu on the
// session bus. Windows uses Shell_NotifyIcon and a Win32 popup menu.
// macOS uses NSStatusItem. On macOS, call [github.com/lewtec/lewkit/x/thread.Bind]
// from main and run [github.com/lewtec/lewkit/x/thread.Loop] so AppKit can
// deliver clicks. Open may be called on that thread or from another goroutine
// while Loop is running.
//
// Icon accepts an image.Image or the bytes of a PNG, JPEG, ICO, or ICNS.
// The driver pads, resizes, and converts to the host format. Name is a
// freedesktop icon name used when Image and Bytes are empty.
package tray

import (
	"context"
	"errors"
	"image"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	// ErrClosed means the tray has been closed.
	ErrClosed = errors.New("tray closed")
	// ErrIcon means the icon bytes are not a PNG, JPEG, ICO, or ICNS this
	// package can read.
	ErrIcon = errors.New("tray icon")
	// ErrNotBound means macOS Open ran before thread.Bind.
	ErrNotBound = errors.New("thread not bound")
	// ErrNotMain means macOS setup did not run on the process main thread.
	ErrNotMain = errors.New("not main thread")
)

// Icon is the image shown in the status area.
// Image wins over Bytes. Name is used only when both are empty.
type Icon struct {
	Image image.Image
	Bytes []byte
	Name  string
}

// Item is one menu row. Children become a submenu.
// OnClick runs in its own goroutine. A Separator ignores Label and OnClick.
type Item struct {
	Label     string
	Disabled  bool
	Checked   bool
	Separator bool
	Children  []Item
	OnClick   func()
}

// Config is the status item. ID defaults to "lewkit".
// Tooltip is the hover text. When Tooltip is empty, Title is used.
type Config struct {
	ID      string
	Title   string
	Tooltip string
	Icon    Icon
	Menu    []Item
}

// Driver opens a host tray.
type Driver interface {
	Open(ctx context.Context, cfg Config) (Tray, error)
}

// Tray is a live status item. Update replaces the title, tooltip, icon, and menu.
type Tray interface {
	Update(cfg Config) error
	Close() error
}

// Open asks the active tray driver for a status item.
// The item is removed when ctx is canceled or Close returns.
func Open(ctx context.Context, cfg Config) (Tray, error) {
	if cfg.ID == "" {
		cfg.ID = "lewkit"
	}
	return driver.WithResult(ctx, func(d Driver) (Tray, error) {
		return d.Open(ctx, cfg)
	})
}

// Tip is the hover string: Tooltip, or Title when Tooltip is empty.
func Tip(cfg Config) string {
	if cfg.Tooltip != "" {
		return cfg.Tooltip
	}
	return cfg.Title
}
