package iosbox

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
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
			t.Errorf("request: %v", err)
			return
		}
		if msg.Op != OpOpen || msg.Target != "https://lew.tec.br" {
			t.Errorf("request %+v", msg)
		}
		if err := os.WriteFile(filepath.Join(dir, "reply.txt"), []byte(askwire.Format(askwire.StatusOK, "")), 0o600); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	got, err := Call(ctx, Request{Op: OpOpen, Target: "https://lew.tec.br"})
	if err != nil {
		t.Fatal(err)
	}
	status, _, err := Result(got)
	if err != nil || status != askwire.StatusOK {
		t.Fatalf("reply %q %v", got, err)
	}
	<-done
}

func TestResult(t *testing.T) {
	_, _, err := Result("error\nnotifications denied")
	if !errors.Is(err, driver.ErrUnavailable) {
		t.Fatal(err)
	}
	status, payload, err := Result("canceled\n")
	if err != nil || status != askwire.StatusCanceled || payload != "" {
		t.Fatalf("%s %q %v", status, payload, err)
	}
}

func TestStateFiles(t *testing.T) {
	dir := t.TempDir()
	ctx := WithDir(t.Context(), dir)
	if err := os.WriteFile(filepath.Join(dir, "battery.json"), []byte(`{"status":"Charging","level":40}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ReadBattery(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "Charging" || got.Level != 40 {
		t.Fatalf("%+v", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "daynight.txt"), []byte("dark\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mode, err := ReadMode(ctx)
	if err != nil || mode != "dark" {
		t.Fatalf("%s %v", mode, err)
	}
}

func TestNoHost(t *testing.T) {
	if Dir() != "" {
		t.Skip()
	}
	_, err := Call(t.Context(), Request{Op: OpNotify})
	if !errors.Is(err, driver.ErrUnavailable) {
		t.Fatal(err)
	}
}
