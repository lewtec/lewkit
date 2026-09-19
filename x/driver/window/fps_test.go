package window

import (
	"testing"
	"time"
)

func TestFPSGet(t *testing.T) {
	var fps FPS
	if v := fps.Get(); v != 0 {
		t.Fatalf("first Get = %v, want 0", v)
	}
	fps.last = time.Now().Add(-100 * time.Millisecond)
	got := fps.Get()
	if got < 5 || got > 20 {
		t.Fatalf("Get after 100ms = %v, want ~10", got)
	}
}
