package d3d12

import "testing"

func TestDispatchGrid(t *testing.T) {
	cases := []struct {
		groups int
		x, y   uint32
	}{
		{1, 1, 1},
		{15000, 15000, 1},
		{65535, 65535, 1},
		{65536, 32768, 2},
		{64800, 64800, 1}, // 1920×1080×4 elements / 128
		{115200, 57600, 2},
		{259200, 64800, 4},
	}
	for _, tc := range cases {
		x, y, err := DispatchGrid(tc.groups)
		if err != nil {
			t.Fatalf("groups %d: %v", tc.groups, err)
		}
		if x != tc.x || y != tc.y {
			t.Fatalf("groups %d: got %d×%d want %d×%d", tc.groups, x, y, tc.x, tc.y)
		}
		if x < 1 || y < 1 || int(x) > maxGroup || int(y) > maxGroup {
			t.Fatalf("groups %d: grid %d×%d outside the dimension limit", tc.groups, x, y)
		}
		if int64(x)*int64(y) < int64(tc.groups) {
			t.Fatalf("groups %d: grid %d×%d does not cover", tc.groups, x, y)
		}
	}
	if _, _, err := DispatchGrid(0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := DispatchGrid(maxGroup*maxGroup + 1); err == nil {
		t.Fatal("expected groups past both dimensions to fail")
	}
}
