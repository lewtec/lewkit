package ios

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/iosbox"
)

func TestNotIOS(t *testing.T) {
	if runtime.GOOS == "ios" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(t.Context())
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatal(err)
	}
}

func TestChoose(t *testing.T) {
	dir := t.TempDir()
	got := make(chan iosbox.Request, 1)
	go func() {
		raw := waitReq(t, dir)
		var msg iosbox.Request
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Errorf("request: %v", err)
			return
		}
		got <- msg
		body := askwire.Format(askwire.StatusOK, `["/cache/inbox/song.mp3"]`)
		if err := os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	paths, err := backend{}.Choose(ctx, filedialog.Request{
		Title:   "Open",
		Filters: []filedialog.Filter{{Name: "Audio", Patterns: []string{"*.mp3"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "/cache/inbox/song.mp3" {
		t.Fatalf("%v", paths)
	}
	msg := <-got
	if msg.Op != iosbox.OpPick || len(msg.Exts) != 1 || msg.Exts[0] != "mp3" {
		t.Fatalf("%+v", msg)
	}
}

func TestCanceled(t *testing.T) {
	dir := t.TempDir()
	go func() {
		_ = waitReq(t, dir)
		body := askwire.Format(askwire.StatusCanceled, "")
		if err := os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	_, err := backend{}.Choose(ctx, filedialog.Request{})
	if !errors.Is(err, filedialog.ErrCanceled) {
		t.Fatal(err)
	}
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
	t.Errorf("no request")
	return nil
}
