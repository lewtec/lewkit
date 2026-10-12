package vulkan

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/present"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
)

func TestMapHoldAndLost(t *testing.T) {
	if !present.Hold(mapErr(drvvulkan.ErrHold)) {
		t.Fatal("hold")
	}
	if !present.Lost(mapErr(drvvulkan.ErrLost)) {
		t.Fatal("lost")
	}
}
