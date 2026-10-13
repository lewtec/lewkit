package hostask

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
)

func TestRequireWithoutHost(t *testing.T) {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		t.Skip()
	}
	t.Cleanup(func() { Set(nil) })
	Set(nil)
	t.Setenv("ELETROCROMO_ASK_DIR", "")
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	err := Require()
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.EqualError(t, err, "component is incompatible: no ask host")
}

func TestRequireWithHook(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip()
	}
	t.Cleanup(func() { Set(nil) })
	Set(func(context.Context, int, string, string) (string, error) { return "", nil })
	require.NoError(t, Require())
}

func TestCallUsesHook(t *testing.T) {
	t.Cleanup(func() { Set(nil) })
	Set(func(context.Context, int, string, string) (string, error) {
		return askwire.Format(askwire.StatusYes, ""), nil
	})
	got, err := Call(t.Context(), KindConfirm, "Delete?", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != askwire.Format(askwire.StatusYes, "") {
		t.Fatalf("reply %q", got)
	}
}

func TestFileCall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELETROCROMO_ASK_DIR", dir)
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("LEWKIT_NO_UI", "")
	if !Available() {
		t.Fatal("ask dir is not available")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		req := filepath.Join(dir, "request.json")
		deadline := time.Now().Add(2 * time.Second)
		var raw []byte
		for time.Now().Before(deadline) {
			var err error
			raw, err = os.ReadFile(req)
			if err == nil && len(raw) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		var msg request
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Errorf("request: %v", err)
			return
		}
		if msg.Kind != KindPrompt || msg.Title != "Name" {
			t.Errorf("request %+v", msg)
		}
		reply := askwire.Format(askwire.StatusOK, "ada")
		if err := os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(reply), 0o600); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	got, err := Call(ctx, KindPrompt, "Name", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != askwire.Format(askwire.StatusOK, "ada") {
		t.Fatalf("reply %q", got)
	}
	<-done
}
