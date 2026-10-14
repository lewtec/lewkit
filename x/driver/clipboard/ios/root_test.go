package ios

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotIOS(t *testing.T) {
	if runtime.GOOS == "ios" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestWriteText(t *testing.T) {
	dir := t.TempDir()
	got := make(chan iosbox.Request, 1)
	go reply(t, dir, got, askwire.Format(askwire.StatusOK, ""))
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	require.NoError(t, (backend{}).WriteText(ctx, "hello"))
	msg := <-got
	if msg.Op != iosbox.OpClipboard || msg.Text == nil || *msg.Text != "hello" {
		require.FailNowf(t, "", "%+v", msg)
	}
}

func TestWriteImage(t *testing.T) {
	dir := t.TempDir()
	got := make(chan iosbox.Request, 1)
	go reply(t, dir, got, askwire.Format(askwire.StatusOK, ""))
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.NRGBA{R: 1, A: 255})
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	require.NoError(t, (backend{}).WriteImage(ctx, img))
	msg := <-got
	raw, err := base64.StdEncoding.DecodeString(msg.PNG)
	if err != nil || len(raw) < 8 || string(raw[:8]) != "\x89PNG\r\n\x1a\n" {
		require.FailNowf(t, "", "png %d %v", len(raw), err)
	}
}

func reply(t *testing.T, dir string, got chan<- iosbox.Request, body string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var raw []byte
	req := filepath.Join(dir, "request.json")
	for time.Now().Before(deadline) {
		next, err := os.ReadFile(req)
		if err == nil && len(next) > 0 {
			raw = next
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	var msg iosbox.Request
	if err := json.Unmarshal(raw, &msg); err != nil {
		assert.Failf(t, "", "request: %v", err)
		return
	}
	got <- msg
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600))
}
