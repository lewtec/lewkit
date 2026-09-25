package dbus

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/media"
)

const playerIface = "org.mpris.MediaPlayer2.Player"

func available(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("%w: session bus: %v", driver.ErrIncompatible, err)
	}
	defer conn.Close()
	names, err := listNames(ctx, conn)
	if err != nil {
		return fmt.Errorf("%w: list dbus names: %v", driver.ErrIncompatible, err)
	}
	for _, name := range names {
		if strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			return nil
		}
	}
	return fmt.Errorf("%w: no MPRIS players found on DBus", driver.ErrIncompatible)
}

func open(ctx context.Context) (media.Driver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	return backend{conn: conn}, nil
}

type backend struct {
	conn *dbus.Conn
}

type playerInfo struct {
	name   string
	status string
	obj    dbus.BusObject
}

func (b backend) getBestPlayer(ctx context.Context) (dbus.BusObject, string, error) {
	names, err := listNames(ctx, b.conn)
	if err != nil {
		return nil, "", err
	}
	var infos []playerInfo
	for _, name := range names {
		if !strings.HasPrefix(name, "org.mpris.MediaPlayer2.") {
			continue
		}
		obj := b.conn.Object(name, "/org/mpris/MediaPlayer2")
		statusVar, err := getProp(ctx, obj, playerIface, "PlaybackStatus")
		if err != nil {
			continue
		}
		status, ok := statusVar.Value().(string)
		if !ok {
			continue
		}
		infos = append(infos, playerInfo{name: name, status: status, obj: obj})
	}
	best, ok := pickPlayer(infos)
	if !ok {
		return nil, "", media.ErrNoPlayer
	}
	return best.obj, best.name, nil
}

func pickPlayer(infos []playerInfo) (playerInfo, bool) {
	if len(infos) == 0 {
		return playerInfo{}, false
	}
	best := infos[0]
	priority := map[string]int{"Playing": 3, "Paused": 2, "Stopped": 1}
	for _, info := range infos[1:] {
		if priority[info.status] > priority[best.status] {
			best = info
		}
	}
	return best, true
}

func (b backend) callAction(ctx context.Context, action string) error {
	obj, _, err := b.getBestPlayer(ctx)
	if err != nil {
		return err
	}
	return obj.CallWithContext(ctx, playerIface+"."+action, 0).Err
}

func (b backend) Next(ctx context.Context) error      { return b.callAction(ctx, "Next") }
func (b backend) Previous(ctx context.Context) error  { return b.callAction(ctx, "Previous") }
func (b backend) PlayPause(ctx context.Context) error { return b.callAction(ctx, "PlayPause") }
func (b backend) Stop(ctx context.Context) error      { return b.callAction(ctx, "Stop") }

func (b backend) GetMetadata(ctx context.Context) (*media.Metadata, error) {
	obj, name, err := b.getBestPlayer(ctx)
	if err != nil {
		return nil, err
	}
	statusVar, err := getProp(ctx, obj, playerIface, "PlaybackStatus")
	if err != nil {
		return nil, err
	}
	status, ok := statusVar.Value().(string)
	if !ok {
		return nil, fmt.Errorf("playback status: unexpected type %T", statusVar.Value())
	}
	metadataVar, err := getProp(ctx, obj, playerIface, "Metadata")
	if err != nil {
		return nil, err
	}
	fields, ok := metadataVar.Value().(map[string]dbus.Variant)
	if !ok {
		return nil, fmt.Errorf("metadata: unexpected type %T", metadataVar.Value())
	}
	res := metadataFrom(name, media.PlaybackStatus(status), fields)
	posVar, err := getProp(ctx, obj, playerIface, "Position")
	if err == nil {
		if pos, ok := int64Of(posVar.Value()); ok {
			res.Position = pos
		}
	}
	return res, nil
}

func metadataFrom(player string, status media.PlaybackStatus, fields map[string]dbus.Variant) *media.Metadata {
	res := &media.Metadata{Player: player, Status: status}
	if v, ok := fields["xesam:title"]; ok {
		if title, ok := v.Value().(string); ok {
			res.Title = title
		}
	}
	if v, ok := fields["xesam:artist"]; ok {
		res.Artist = artistOf(v.Value())
	}
	if v, ok := fields["mpris:artUrl"]; ok {
		if artURL, ok := v.Value().(string); ok {
			res.ArtUrl = artURL
		}
	}
	if v, ok := fields["mpris:length"]; ok {
		if length, ok := int64Of(v.Value()); ok {
			res.Length = length
		}
	}
	return res
}

func artistOf(value any) string {
	switch val := value.(type) {
	case []string:
		return strings.Join(val, ", ")
	case []any:
		var artists []string
		for _, item := range val {
			if text, ok := item.(string); ok {
				artists = append(artists, text)
			}
		}
		return strings.Join(artists, ", ")
	case string:
		return val
	default:
		return ""
	}
}

func int64Of(value any) (int64, bool) {
	switch val := value.(type) {
	case int64:
		return val, true
	case uint64:
		return int64(val), true
	default:
		return 0, false
	}
}

func (b backend) Watch(ctx context.Context, callback func(*media.Metadata)) error {
	rule := "type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged',path='/org/mpris/MediaPlayer2'"
	if err := b.conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.AddMatch", 0, rule).Err; err != nil {
		return err
	}
	signals := make(chan *dbus.Signal, 10)
	b.conn.Signal(signals)
	defer b.conn.RemoveSignal(signals)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case signal := <-signals:
			if signal == nil || len(signal.Body) < 2 {
				continue
			}
			iface, ok := signal.Body[0].(string)
			if !ok || iface != playerIface {
				continue
			}
			changed, ok := signal.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}
			if _, ok := changed["Metadata"]; !ok {
				continue
			}
			meta, err := b.GetMetadata(ctx)
			if err == nil {
				callback(meta)
			}
		}
	}
}

func listNames(ctx context.Context, conn *dbus.Conn) ([]string, error) {
	var names []string
	err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names)
	return names, err
}

func getProp(ctx context.Context, obj dbus.BusObject, iface, name string) (dbus.Variant, error) {
	var value dbus.Variant
	err := obj.CallWithContext(ctx, "org.freedesktop.DBus.Properties.Get", 0, iface, name).Store(&value)
	return value, err
}
