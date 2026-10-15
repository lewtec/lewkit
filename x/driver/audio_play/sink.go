package audio_play

import (
	"context"
	"errors"
)

// ErrSink means Config.Sink matched no listed device.
var ErrSink = errors.New("sound sink")

// MapSinks lists devices after ctx is checked and copies ID and name onto Sink.
func MapSinks[T any](ctx context.Context, list func() ([]T, error), id, name func(T) string) ([]Sink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items, err := list()
	if err != nil {
		return nil, err
	}
	out := make([]Sink, len(items))
	for i, item := range items {
		out[i] = Sink{ID: id(item), Name: name(item)}
	}
	return out, nil
}

// FindSink returns the device whose ID or name equals want.
// An ID match wins over a name match.
func FindSink(want string, sinks []Sink) (Sink, error) {
	for _, sink := range sinks {
		if sink.ID == want {
			return sink, nil
		}
	}
	for _, sink := range sinks {
		if sink.Name == want && sink.Name != "" {
			return sink, nil
		}
	}
	return Sink{}, ErrSink
}
