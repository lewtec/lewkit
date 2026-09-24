//go:build linux

package portal

import (
	"context"
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/lewtec/lewkit/x/event"
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

type source struct {
	mu     sync.Mutex
	scheme appearance.Scheme
	bus    *event.Bus[appearance.Scheme]
}

var (
	openOnce sync.Once
	openSrc  *source
	openErr  error
)

func open(ctx context.Context) (appearance.Driver, error) {
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
		openSrc = &source{scheme: scheme, bus: event.New[appearance.Scheme]()}
		go openSrc.loop(conn, signals)
	})
	if openErr != nil {
		return nil, openErr
	}
	return openSrc, nil
}

func (src *source) loop(conn *dbus.Conn, signals chan *dbus.Signal) {
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
		src.mu.Lock()
		same := src.scheme == scheme
		src.scheme = scheme
		src.mu.Unlock()
		if !same {
			src.bus.Publish(scheme)
		}
	}
}

func (src *source) Current(context.Context) (appearance.Scheme, error) {
	src.mu.Lock()
	defer src.mu.Unlock()
	return src.scheme, nil
}

func (src *source) Watch(ctx context.Context) (<-chan appearance.Scheme, error) {
	src.mu.Lock()
	current := src.scheme
	src.mu.Unlock()
	return appearance.Changes(ctx, current, src.bus.Subscribe(ctx)), nil
}

func readScheme(conn *dbus.Conn) (appearance.Scheme, error) {
	var value dbus.Variant
	err := conn.Object(busName, path).Call(iface+".Read", 0, namespace, key).Store(&value)
	if err != nil {
		return appearance.Light, err
	}
	scheme, ok := schemeOf(value)
	if !ok {
		return appearance.Light, fmt.Errorf("%w: color-scheme type", driver.ErrUnavailable)
	}
	return scheme, nil
}

func schemeOf(value any) (appearance.Scheme, bool) {
	switch typed := value.(type) {
	case dbus.Variant:
		return schemeOf(typed.Value())
	case uint32:
		return appearance.FromPortal(typed), true
	case uint16:
		return appearance.FromPortal(uint32(typed)), true
	case int32:
		if typed < 0 {
			return appearance.Light, false
		}
		return appearance.FromPortal(uint32(typed)), true
	case int16:
		if typed < 0 {
			return appearance.Light, false
		}
		return appearance.FromPortal(uint32(typed)), true
	case byte:
		return appearance.FromPortal(uint32(typed)), true
	default:
		return appearance.Light, false
	}
}
