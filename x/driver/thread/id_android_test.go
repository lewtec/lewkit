//go:build android

package thread

import "testing"

func TestOSThreadStable(t *testing.T) {
	id := OSThread()
	if id == 0 {
		t.Fatal("tid is 0")
	}
	if again := OSThread(); again != id {
		t.Fatalf("tid %d then %d", id, again)
	}
}
