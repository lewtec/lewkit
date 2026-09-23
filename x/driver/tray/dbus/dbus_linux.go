//go:build linux

package dbus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"

	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/lewtec/lewkit/x/image/convert"
)

const (
	itemPath        = dbus.ObjectPath("/StatusNotifierItem")
	menuPath        = dbus.ObjectPath("/MenuBar")
	itemInterface   = "org.kde.StatusNotifierItem"
	menuInterface   = "com.canonical.dbusmenu"
	watcherService  = "org.kde.StatusNotifierWatcher"
	watcherPath     = dbus.ObjectPath("/StatusNotifierWatcher")
	watcherRegister = watcherService + ".RegisterStatusNotifierItem"
)

var serviceSerial atomic.Uint32

type opener struct{}

func (opener) Open(ctx context.Context, cfg tray.Config) (tray.Tray, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("session bus: %w", err)
	}
	item := &statusItem{conn: conn, cfg: cfg}
	if _, err := iconPixmaps(cfg); err != nil {
		return nil, abandon(conn, err)
	}
	if err := item.export(); err != nil {
		return nil, abandon(conn, err)
	}
	name := fmt.Sprintf("org.kde.StatusNotifierItem-%d-%d", os.Getpid(), serviceSerial.Add(1))
	reply, err := conn.RequestName(name, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, abandon(conn, fmt.Errorf("request name: %w", err))
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, abandon(conn, fmt.Errorf("%w: %s", errNameTaken, name))
	}
	item.busName = name
	watcher := conn.Object(watcherService, watcherPath)
	call := watcher.Call(watcherRegister, 0, name)
	if call.Err != nil {
		return nil, abandon(conn, fmt.Errorf("register status notifier: %w", call.Err))
	}
	if err := conn.Emit(itemPath, itemInterface+".NewStatus", "Active"); err != nil {
		return nil, abandon(conn, err)
	}
	if err := conn.Emit(itemPath, itemInterface+".NewIcon"); err != nil {
		return nil, abandon(conn, err)
	}
	if err := conn.Emit(itemPath, itemInterface+".NewMenu"); err != nil {
		return nil, abandon(conn, err)
	}
	context.AfterFunc(ctx, func() {
		if err := item.Close(); err != nil {
			return
		}
	})
	return item, nil
}

var errNameTaken = errors.New("tray name taken")

func abandon(conn *dbus.Conn, err error) error {
	if closeErr := conn.Close(); closeErr != nil {
		return errors.Join(err, closeErr)
	}
	return err
}

type pixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

type toolTip struct {
	Icon        string
	Image       []pixmap
	Title       string
	Description string
}

type statusItem struct {
	mu       sync.Mutex
	conn     *dbus.Conn
	cfg      tray.Config
	menu     *menuNode
	props    *prop.Properties
	busName  string
	closed   bool
	closeOne sync.Once
}

func (item *statusItem) export() error {
	notifier := &notifier{item: item}
	if err := item.conn.Export(notifier, itemPath, itemInterface); err != nil {
		return fmt.Errorf("export status notifier: %w", err)
	}
	item.menu = &menuNode{item: item, revision: 1}
	if err := item.conn.Export(item.menu, menuPath, menuInterface); err != nil {
		return fmt.Errorf("export menu: %w", err)
	}
	if err := exportNode(item.conn, itemPath, notifierSpec()); err != nil {
		return err
	}
	if err := exportNode(item.conn, menuPath, menuSpec()); err != nil {
		return err
	}
	props, err := prop.Export(item.conn, itemPath, prop.Map{
		itemInterface: propMap(item.properties()),
	})
	if err != nil {
		return fmt.Errorf("export properties: %w", err)
	}
	item.props = props
	_, err = prop.Export(item.conn, menuPath, prop.Map{
		menuInterface: propMap(map[string]any{
			"Version":       uint32(3),
			"TextDirection": "ltr",
			"Status":        "normal",
			"IconThemePath": []string{},
		}),
	})
	if err != nil {
		return fmt.Errorf("export menu properties: %w", err)
	}
	return nil
}

func (item *statusItem) properties() map[string]any {
	return map[string]any{
		"Category":            "ApplicationStatus",
		"Id":                  item.cfg.ID,
		"Title":               item.cfg.Title,
		"Status":              "Active",
		"WindowId":            int32(0),
		"IconThemePath":       "",
		"Menu":                menuPath,
		"ItemIsMenu":          true,
		"IconName":            iconName(item.cfg),
		"IconPixmap":          mustPixmaps(item.cfg),
		"OverlayIconName":     "",
		"OverlayIconPixmap":   []pixmap{},
		"AttentionIconName":   "",
		"AttentionIconPixmap": []pixmap{},
		"ToolTip":             toolTip{Title: item.cfg.Title, Description: item.cfg.Tooltip},
	}
}

func (item *statusItem) Update(cfg tray.Config) error {
	if cfg.ID == "" {
		cfg.ID = item.cfg.ID
	}
	item.mu.Lock()
	defer item.mu.Unlock()
	if item.closed {
		return tray.ErrClosed
	}
	pixmaps, err := iconPixmaps(cfg)
	if err != nil {
		return err
	}
	item.cfg = cfg
	item.props.SetMust(itemInterface, "Id", cfg.ID)
	item.props.SetMust(itemInterface, "Title", cfg.Title)
	item.props.SetMust(itemInterface, "IconName", iconName(cfg))
	item.props.SetMust(itemInterface, "IconPixmap", pixmaps)
	item.props.SetMust(itemInterface, "ToolTip", toolTip{Title: cfg.Title, Description: cfg.Tooltip})
	item.menu.revision++
	revision := item.menu.revision
	if err := item.conn.Emit(itemPath, itemInterface+".NewTitle"); err != nil {
		return err
	}
	if err := item.conn.Emit(itemPath, itemInterface+".NewIcon"); err != nil {
		return err
	}
	if err := item.conn.Emit(itemPath, itemInterface+".NewToolTip"); err != nil {
		return err
	}
	return item.conn.Emit(menuPath, menuInterface+".LayoutUpdated", revision, int32(0))
}

func (item *statusItem) Close() error {
	var err error
	item.closeOne.Do(func() {
		item.mu.Lock()
		item.closed = true
		conn := item.conn
		name := item.busName
		item.mu.Unlock()
		if conn == nil {
			return
		}
		if name != "" {
			if _, releaseErr := conn.ReleaseName(name); releaseErr != nil {
				err = releaseErr
			}
		}
		if closeErr := conn.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	})
	return err
}

func (item *statusItem) snapshot() tray.Config {
	item.mu.Lock()
	defer item.mu.Unlock()
	return item.cfg
}

func iconName(cfg tray.Config) string {
	if cfg.Icon.Image != nil || len(cfg.Icon.Bytes) != 0 {
		return ""
	}
	return cfg.Icon.Name
}

func mustPixmaps(cfg tray.Config) []pixmap {
	pixmaps, err := iconPixmaps(cfg)
	if err != nil || pixmaps == nil {
		return []pixmap{}
	}
	return pixmaps
}

func iconPixmaps(cfg tray.Config) ([]pixmap, error) {
	src, err := tray.Source(cfg.Icon)
	if err != nil || src == nil {
		return []pixmap{}, err
	}
	images, err := convert.Squares(src, convert.LinuxSizes)
	if err != nil || len(images) == 0 {
		return []pixmap{}, err
	}
	out := make([]pixmap, 0, len(images))
	for _, img := range images {
		out = append(out, pixmap{
			Width:  int32(img.Bounds().Dx()),
			Height: int32(img.Bounds().Dy()),
			Data:   convert.ARGB(img),
		})
	}
	return out, nil
}

type notifier struct {
	item *statusItem
}

func (notifier) ContextMenu(int32, int32) *dbus.Error       { return nil }
func (notifier) Activate(int32, int32) *dbus.Error          { return nil }
func (notifier) SecondaryActivate(int32, int32) *dbus.Error { return nil }
func (notifier) Scroll(int32, string) *dbus.Error           { return nil }

func propMap(values map[string]any) map[string]*prop.Prop {
	out := make(map[string]*prop.Prop, len(values))
	for name, value := range values {
		out[name] = &prop.Prop{Value: value, Emit: prop.EmitTrue}
	}
	return out
}

func exportNode(conn *dbus.Conn, path dbus.ObjectPath, spec introspect.Interface) error {
	node := &introspect.Node{
		Name: string(path),
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			prop.IntrospectData,
			spec,
		},
	}
	if err := conn.Export(introspect.NewIntrospectable(node), path, "org.freedesktop.DBus.Introspectable"); err != nil {
		return fmt.Errorf("export introspectable: %w", err)
	}
	return nil
}
