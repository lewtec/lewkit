package host

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/hostask"
)

func TestNeedsHost(t *testing.T) {
	t.Setenv("ELETROCROMO_ASK_DIR", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_CACHE_DIR", "")
	t.Setenv("LEWKIT_CACHE_DIR", "")
	hostask.Set(nil)
	t.Cleanup(func() { hostask.Set(nil) })
	if err := (base{}).CheckCompatibility(t.Context()); err == nil {
		t.Fatal("host launcher is compatible without an ask host")
	}
	hostask.Set(func(context.Context, int, string, string) (string, error) {
		return askwire.Format(askwire.StatusYes, ""), nil
	})
	if err := (base{}).CheckCompatibility(t.Context()); err != nil {
		t.Fatal(err)
	}
	ok, err := backend{}.Confirm(t.Context(), "Delete?")
	if err != nil || !ok {
		t.Fatalf("confirm %v %v", ok, err)
	}
}
