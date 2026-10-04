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
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/driver/notification"
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
			t.Errorf("request: %v", err)
			return
		}
		got <- msg
		body := askwire.Format(askwire.StatusOK, "")
		if err := os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(body), 0o600); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(iosbox.WithDir(t.Context(), dir), 2*time.Second)
	defer cancel()
	err := backend{}.Notify(ctx, notification.Notification{
		Title:   "Hello",
		Message: "done",
		Urgency: "critical",
		ID:      notification.StatusID,
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := <-got
	if msg.Op != iosbox.OpNotify || msg.Title != "Hello" || msg.Urgency != "critical" || msg.ID != notification.StatusID {
		t.Fatalf("%+v", msg)
	}
}
