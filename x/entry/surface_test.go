package entry

import "testing"

func TestDeliverPointerReachesTheHandler(t *testing.T) {
	var gotX, gotY, gotAction int
	HandlePointer(func(x, y, action int) {
		gotX, gotY, gotAction = x, y, action
	})
	DeliverPointer(3, 4, 2)
	if gotX != 3 || gotY != 4 || gotAction != 2 {
		t.Fatalf("pointer = %d %d %d", gotX, gotY, gotAction)
	}
	HandleResize(func(width, height int) {
		gotX, gotY = width, height
	})
	DeliverResize(8, 9)
	if gotX != 8 || gotY != 9 {
		t.Fatalf("resize = %d %d", gotX, gotY)
	}
	lost := false
	HandleSurfaceLost(func() { lost = true })
	DeliverSurfaceLost()
	if !lost {
		t.Fatal("surface lost was not delivered")
	}
}
