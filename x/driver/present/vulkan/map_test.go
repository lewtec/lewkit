package vulkan

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver/present"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/stretchr/testify/require"
)

func TestMapHoldAndLost(t *testing.T) {
	if !present.Hold(mapErr(drvvulkan.ErrHold)) {
		require.FailNow(t, fmt.Sprint("hold"))
	}
	if !present.Lost(mapErr(drvvulkan.ErrLost)) {
		require.FailNow(t, fmt.Sprint("lost"))
	}
}
