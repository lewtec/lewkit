package sound

import "errors"

// ErrSink means Config.Sink matched no listed device.
var ErrSink = errors.New("sound sink")

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
