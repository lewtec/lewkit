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
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/driver/notification"
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

func TestNotify(t *testing.T) {
	dir := t.TempDir()
	got := make(chan iosbox.Request, 1)
	go func() {
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
		body := askwire.Format(askwire.StatusOK, "")
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600))
	}()
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	err := backend{}.Notify(ctx, notification.Notification{
		Title:   "Hello",
		Message: "done",
		Urgency: "critical",
		ID:      notification.StatusID,
	})
	require.NoError(t, err)
	msg := <-got
	if msg.Op != iosbox.OpNotify || msg.Title != "Hello" || msg.Urgency != "critical" || msg.ID != notification.StatusID {
		require.FailNowf(t, "", "%+v", msg)
	}
}
