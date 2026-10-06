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

func TestDeliverInsetsReplaysTheLatest(t *testing.T) {
	DeliverInsets(1, 2, 3, 4, 50, 80)
	var got [6]int
	HandleInsets(func(left, top, right, bottom, width, height int) {
		got = [6]int{left, top, right, bottom, width, height}
	})
	if got != [6]int{1, 2, 3, 4, 50, 80} {
		t.Fatalf("replay = %v", got)
	}
	DeliverInsets(0, 9, 0, 1, 50, 80)
	if got != [6]int{0, 9, 0, 1, 50, 80} {
		t.Fatalf("update = %v", got)
	}
}
