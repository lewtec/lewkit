package ios

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotIOS(t *testing.T) {
	if runtime.GOOS == "ios" {
		test.Skip(t)
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestChoose(t *testing.T) {
	dir := t.TempDir()
	got := make(chan iosbox.Request, 1)
	go func() {
		raw := waitReq(t, dir)
		var msg iosbox.Request
		if err := json.Unmarshal(raw, &msg); err != nil {
			assert.Failf(t, "", "request: %v", err)
			return
		}
		got <- msg
		body := askwire.Format(askwire.StatusOK, `["/cache/inbox/song.mp3"]`)
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600))
	}()
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	paths, err := backend{}.Choose(ctx, filedialog.Request{
		Title:   "Open",
		Filters: []filedialog.Filter{{Name: "Audio", Patterns: []string{"*.mp3"}}},
	})
	require.NoError(t, err)
	if len(paths) != 1 || paths[0] != "/cache/inbox/song.mp3" {
		require.FailNowf(t, "", "%v", paths)
	}
	msg := <-got
	if msg.Op != iosbox.OpPick || len(msg.Exts) != 1 || msg.Exts[0] != "mp3" {
		require.FailNowf(t, "", "%+v", msg)
	}
}

func TestCanceled(t *testing.T) {
	dir := t.TempDir()
	go func() {
		_ = waitReq(t, dir)
		body := askwire.Format(askwire.StatusCanceled, "")
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600))
	}()
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	_, err := backend{}.Choose(ctx, filedialog.Request{})
	require.ErrorIs(t, err, filedialog.ErrCanceled)
}

func waitReq(t *testing.T, dir string) []byte {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	req := filepath.Join(dir, "request.json")
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(req)
		if err == nil && len(raw) > 0 {
			return raw
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Failf(t, "", "no request")
	return nil
}
