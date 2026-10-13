package cache

import (
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
)

func TestPublish(t *testing.T) {
	live := New(daynight.Light)
	got, err := live.Current(t.Context())
	if err != nil || got != daynight.Light {
		t.Fatalf("current %v %v", got, err)
	}
	changes, err := live.Watch(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if scheme := <-changes; scheme != daynight.Light {
		t.Fatalf("watch %v", scheme)
	}
	live.Publish(daynight.Light)
	live.Publish(daynight.Dark)
	select {
	case scheme := <-changes:
		if scheme != daynight.Dark {
			t.Fatalf("publish %v", scheme)
		}
	case <-time.After(time.Second):
		t.Fatal("scheme did not change")
	}
}
