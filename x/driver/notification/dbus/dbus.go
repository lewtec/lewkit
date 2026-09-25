package dbus

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
)

type backend struct {
	conn *dbus.Conn
	mu   sync.Mutex
	ids  map[uint32]uint32
}

func (b *backend) Notify(ctx context.Context, n notification.Notification) error {
	obj := b.conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
	b.mu.Lock()
	replaces := n.ID
	if actual, ok := b.ids[n.ID]; ok && n.ID != 0 {
		replaces = actual
	}
	b.mu.Unlock()

	urgency := byte(1)
	switch n.Urgency {
	case "low":
		urgency = 0
	case "critical":
		urgency = 2
	}
	hints := map[string]dbus.Variant{
		"urgency": dbus.MakeVariant(urgency),
	}
	if n.HasProgress {
		hints["value"] = dbus.MakeVariant(int32(n.Progress * 100))
	}

	call := obj.CallWithContext(ctx, "org.freedesktop.Notifications.Notify", 0,
		"lewkit", replaces, n.Icon, n.Title, n.Message, []string{}, hints, int32(-1))
	var serverID uint32
	if err := call.Store(&serverID); err != nil {
		return err
	}
	if n.ID != 0 {
		b.mu.Lock()
		b.ids[n.ID] = serverID
		b.mu.Unlock()
	}
	return nil
}

func open(ctx context.Context) (notification.Driver, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	return &backend{conn: conn, ids: map[uint32]uint32{}}, nil
}

func available(ctx context.Context) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return fmt.Errorf("%w: session bus: %w", driver.ErrIncompatible, err)
	}
	var names []string
	err = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return fmt.Errorf("%w: list names: %w", driver.ErrIncompatible, err)
	}
	if slices.Contains(names, "org.freedesktop.Notifications") {
		return nil
	}
	err = conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.StartServiceByName", 0,
		"org.freedesktop.Notifications", uint32(0)).Err
	if err != nil {
		return fmt.Errorf("%w: notifications service: %w", driver.ErrIncompatible, err)
	}
	return nil
}
