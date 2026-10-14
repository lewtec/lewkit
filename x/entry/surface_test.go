package entry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeliverPointerReachesTheHandler(t *testing.T) {
	var gotX, gotY, gotAction int
	HandlePointer(func(x, y, action int) {
		gotX, gotY, gotAction = x, y, action
	})
	DeliverPointer(3, 4, 2)
	if gotX != 3 || gotY != 4 || gotAction != 2 {
		require.FailNowf(t, "", "pointer = %d %d %d", gotX, gotY, gotAction)
	}
	HandleResize(func(width, height int) {
		gotX, gotY = width, height
	})
	DeliverResize(8, 9)
	if gotX != 8 || gotY != 9 {
		require.FailNowf(t, "", "resize = %d %d", gotX, gotY)
	}
	lost := false
	HandleSurfaceLost(func() { lost = true })
	DeliverSurfaceLost()
	if !lost {
		require.FailNow(t, fmt.Sprint("surface lost was not delivered"))
	}
}

func TestDeliverInsetsReplaysTheLatest(t *testing.T) {
	DeliverInsets(1, 2, 3, 4, 50, 80)
	var got [6]int
	HandleInsets(func(left, top, right, bottom, width, height int) {
		got = [6]int{left, top, right, bottom, width, height}
	})
	if got != [6]int{1, 2, 3, 4, 50, 80} {
		require.FailNowf(t, "", "replay = %v", got)
	}
	DeliverInsets(0, 9, 0, 1, 50, 80)
	if got != [6]int{0, 9, 0, 1, 50, 80} {
		require.FailNowf(t, "", "update = %v", got)
	}
}
