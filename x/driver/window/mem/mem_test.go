package mem

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

func TestAppModeDisablesMemoryWindow(t *testing.T) {
	driver.SetAppMode(true)
	t.Cleanup(func() { driver.SetAppMode(false) })
	err := factory{}.CheckCompatibility(t.Context())
	if err == nil {
		t.Fatal("memory window stayed compatible in an app")
	}
	if _, err := window.Open(t.Context(), window.Config{Width: 8, Height: 8}); err == nil {
		t.Fatal("app opened a memory window")
	}
}
