package hostask

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/test"
)

func TestRequireWithoutHost(t *testing.T) {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		test.Skip(t)
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
		test.Skip(t)
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
	require.NoError(t, err)
	if got != askwire.Format(askwire.StatusYes, "") {
		require.FailNowf(t, "", "reply %q", got)
	}
}

func TestFileCall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ELETROCROMO_ASK_DIR", dir)
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("LEWKIT_NO_UI", "")
	if !Available() {
		require.FailNow(t, fmt.Sprint("ask dir is not available"))
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
			assert.Failf(t, "", "request: %v", err)
			return
		}
		if msg.Kind != KindPrompt || msg.Title != "Name" {
			assert.Failf(t, "", "request %+v", msg)
		}
		reply := askwire.Format(askwire.StatusOK, "ada")
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(reply), 0o600))
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	got, err := Call(ctx, KindPrompt, "Name", "")
	require.NoError(t, err)
	if got != askwire.Format(askwire.StatusOK, "ada") {
		require.FailNowf(t, "", "reply %q", got)
	}
	<-done
}
