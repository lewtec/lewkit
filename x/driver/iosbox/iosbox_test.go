package iosbox

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/internal/uikitapp"
	"github.com/lewtec/lewkit/x/test"
)

func TestCall(t *testing.T) {
	dir := t.TempDir()
	ctx := WithDir(t.Context(), dir)
	done := make(chan struct{})
	go func() {
		defer close(done)
		req := filepath.Join(dir, "request.json")
		deadline := time.Now().Add(2 * time.Second)
		var raw []byte
		for time.Now().Before(deadline) {
			got, err := os.ReadFile(req)
			if err == nil && len(got) > 0 {
				raw = got
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		var msg Request
		if err := json.Unmarshal(raw, &msg); err != nil {
			assert.Failf(t, "", "request: %v", err)
			return
		}
		if msg.Op != OpOpen || msg.Target != "https://lew.tec.br" {
			assert.Failf(t, "", "request %+v", msg)
		}
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(askwire.Format(askwire.StatusOK, "")), 0o600))
	}()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	got, err := Call(ctx, Request{Op: OpOpen, Target: "https://lew.tec.br"})
	require.NoError(t, err)
	status, _, err := Result(got)
	if err != nil || status != askwire.StatusOK {
		require.FailNowf(t, "", "reply %q %v", got, err)
	}
	<-done
}

func TestResult(t *testing.T) {
	_, _, err := Result("error\nnotifications denied")
	require.ErrorIs(t, err, driver.ErrUnavailable)
	status, payload, err := Result("canceled\n")
	if err != nil || status != askwire.StatusCanceled || payload != "" {
		require.FailNowf(t, "", "%s %q %v", status, payload, err)
	}
}

func TestStateFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := WithDir(t.Context(), dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "battery.json"), []byte(`{"status":"Charging","level":40}`), 0o600))
	got, err := ReadBattery(ctx)
	require.NoError(t, err)
	if got.Status != "Charging" || got.Level != 40 {
		require.FailNowf(t, "", "%+v", got)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "daynight.txt"), []byte("dark\n"), 0o600))
	mode, err := ReadMode(ctx)
	if err != nil || mode != "dark" {
		require.FailNowf(t, "", "%s %v", mode, err)
	}
}

func TestRequire(t *testing.T) {
	if runtime.GOOS == "ios" && (uikitapp.OwnsUI() || Dir() != "") {
		require.NoError(t, Require())
		return
	}
	require.ErrorIs(t, Require(), driver.ErrIncompatible)
}

func TestNoHost(t *testing.T) {
	if runtime.GOOS == "ios" || Dir() != "" {
		test.Skip(t)
	}
	_, err := Call(t.Context(), Request{Op: OpNotify})
	require.ErrorIs(t, err, driver.ErrUnavailable)
}
