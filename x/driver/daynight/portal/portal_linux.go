//go:build linux && !android

package portal

import (
	"context"
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/daynight/internal/cache"
)

const (
	busName   = "org.freedesktop.portal.Desktop"
	path      = dbus.ObjectPath("/org/freedesktop/portal/desktop")
	iface     = "org.freedesktop.portal.Settings"
	namespace = "org.freedesktop.appearance"
	key       = "color-scheme"
)

func available(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	defer conn.Close()
	if _, err := readScheme(conn); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

var (
	openOnce sync.Once
	openSrc  *cache.Cache
	openErr  error
)

func open(ctx context.Context) (daynight.Driver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	openOnce.Do(func() {
		conn, err := dbus.ConnectSessionBus()
		if err != nil {
			openErr = err
			return
		}
		scheme, err := readScheme(conn)
		if err != nil {
			conn.Close()
			openErr = err
			return
		}
		if err := conn.AddMatchSignal(
			dbus.WithMatchObjectPath(path),
			dbus.WithMatchInterface(iface),
			dbus.WithMatchMember("SettingChanged"),
		); err != nil {
			conn.Close()
			openErr = err
			return
		}
		signals := make(chan *dbus.Signal, 16)
		conn.Signal(signals)
		openSrc = cache.New(scheme)
		go loop(openSrc, conn, signals)
	})
	if openErr != nil {
		return nil, openErr
	}
	return openSrc, nil
}

func loop(src *cache.Cache, conn *dbus.Conn, signals chan *dbus.Signal) {
	defer conn.Close()
	for signal := range signals {
		if signal == nil || len(signal.Body) < 3 {
			continue
		}
		gotNamespace, _ := signal.Body[0].(string)
		gotKey, _ := signal.Body[1].(string)
		if gotNamespace != namespace || gotKey != key {
			continue
		}
		scheme, ok := schemeOf(signal.Body[2])
		if !ok {
			continue
		}
		src.Publish(scheme)
	}
}

func readScheme(conn *dbus.Conn) (daynight.Mode, error) {
	var value dbus.Variant
	err := conn.Object(busName, path).Call(iface+".Read", 0, namespace, key).Store(&value)
	if err != nil {
		return daynight.Light, err
	}
	scheme, ok := schemeOf(value)
	if !ok {
		return daynight.Light, fmt.Errorf("%w: color-scheme type", driver.ErrUnavailable)
	}
	return scheme, nil
}

// fromPortal maps the portal color-scheme value.
// 1 is prefer-dark. 0 (no preference) and 2 (prefer-light) are light.
func fromPortal(value uint32) daynight.Mode {
	if value == 1 {
		return daynight.Dark
	}
	return daynight.Light
}

func schemeOf(value any) (daynight.Mode, bool) {
	switch typed := value.(type) {
	case dbus.Variant:
		return schemeOf(typed.Value())
	case uint32:
		return fromPortal(typed), true
	case uint16:
		return fromPortal(uint32(typed)), true
	case int32:
		if typed < 0 {
			return daynight.Light, false
		}
		return fromPortal(uint32(typed)), true
	case int16:
		if typed < 0 {
			return daynight.Light, false
		}
		return fromPortal(uint32(typed)), true
	case byte:
		return fromPortal(uint32(typed)), true
	default:
		return daynight.Light, false
	}
}
