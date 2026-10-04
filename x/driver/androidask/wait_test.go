package androidask

import "testing"

func TestDeliverOnce(t *testing.T) {
	ch, g, err := begin()
	if err != nil {
		t.Fatal(err)
	}
	defer end()
	deliver(g, "ok\n")
	deliver(g, "again")
	got := <-ch
	if got != "ok\n" {
		t.Fatalf("got %q", got)
	}
	if _, _, err := begin(); err == nil {
		t.Fatal("second begin succeeded")
	}
}
