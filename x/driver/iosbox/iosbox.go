// Package iosbox asks the packaged iOS host to perform one system call.
//
// The host watches ELETROCROMO_IOS_DIR, or <cache>/ios when GOOS is ios.
// Go writes request.json and waits for reply.txt. The reply is an askwire
// message. battery.json and daynight.txt are files the host refreshes.
// They are not requests. WithDir selects a directory for one context.
//
// Ops are clipboard, notify, open, and pick. The host presents the UI.
package iosbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
)

const (
	// OpClipboard writes the pasteboard.
	OpClipboard = "clipboard"
	// OpNotify posts a local notification.
	OpNotify = "notify"
	// OpOpen opens a URL or file.
	OpOpen = "open"
	// OpPick shows the document picker.
	OpPick = "pick"
)

// Request is one host call.
// Text is a pointer so an empty clipboard write is still sent.
type Request struct {
	Op        string   `json:"op"`
	Text      *string  `json:"text,omitempty"`
	PNG       string   `json:"png,omitempty"`
	Title     string   `json:"title,omitempty"`
	Message   string   `json:"message,omitempty"`
	Urgency   string   `json:"urgency,omitempty"`
	ID        uint32   `json:"id,omitempty"`
	Target    string   `json:"target,omitempty"`
	Save      bool     `json:"save,omitempty"`
	Folder    bool     `json:"folder,omitempty"`
	Multiple  bool     `json:"multiple,omitempty"`
	Name      string   `json:"name,omitempty"`
	Directory string   `json:"directory,omitempty"`
	Exts      []string `json:"exts,omitempty"`
}

// Battery is the host battery file.
// Level is 0..100. A negative level has no reading.
type Battery struct {
	Status string `json:"status"`
	Level  int    `json:"level"`
}

type dirKey struct{}

// WithDir points Call and the state reads at dir.
// A test uses it so packages can run in parallel. The host still uses Dir.
func WithDir(ctx context.Context, dir string) context.Context {
	return context.WithValue(ctx, dirKey{}, dir)
}

// Dir is the directory the host watches.
// ELETROCROMO_IOS_DIR wins. On ios, <cache>/ios is the fallback.
func Dir() string {
	if dir := os.Getenv("ELETROCROMO_IOS_DIR"); dir != "" {
		return dir
	}
	if runtime.GOOS != "ios" {
		return ""
	}
	cache := os.Getenv("ELETROCROMO_CACHE_DIR")
	if cache == "" {
		cache = os.Getenv("LEWKIT_CACHE_DIR")
	}
	if cache == "" {
		return ""
	}
	return filepath.Join(cache, "ios")
}

func dirFrom(ctx context.Context) string {
	if ctx != nil {
		if dir, _ := ctx.Value(dirKey{}).(string); dir != "" {
			return dir
		}
	}
	return Dir()
}

// Available reports whether a host directory is set.
func Available() bool { return Dir() != "" }

var fileMu sync.Mutex

// Call writes req and waits for the host reply.
func Call(ctx context.Context, req Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dir := dirFrom(ctx)
	if dir == "" {
		return "", fmt.Errorf("%w: no ios host", driver.ErrUnavailable)
	}
	fileMu.Lock()
	defer fileMu.Unlock()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	reqPath := filepath.Join(dir, "request.json")
	replyPath := filepath.Join(dir, "reply.txt")
	_ = os.Remove(replyPath)
	payload, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	tmp := reqPath + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, reqPath); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	defer func() { _ = os.Remove(reqPath) }()
	defer func() { _ = os.Remove(replyPath) }()

	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", context.Cause(ctx)
		case <-tick.C:
			raw, err := os.ReadFile(replyPath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return "", err
			}
			if len(raw) == 0 {
				continue
			}
			return string(raw), nil
		}
	}
}

// Result splits a host reply.
// OK and canceled are not errors. Any other status is unavailable.
func Result(raw string) (status, payload string, err error) {
	status, payload = askwire.Split(raw)
	payload = strings.TrimRight(payload, "\n")
	switch status {
	case askwire.StatusOK, askwire.StatusCanceled:
		return status, payload, nil
	default:
		msg := strings.TrimSpace(payload)
		if msg == "" {
			msg = status
		}
		if msg == "" {
			msg = "ios host"
		}
		return status, payload, fmt.Errorf("%w: %s", driver.ErrUnavailable, msg)
	}
}

// ReadBattery reads the host battery file.
func ReadBattery(ctx context.Context) (Battery, error) {
	raw, err := readNamed(ctx, "battery.json")
	if err != nil {
		return Battery{}, err
	}
	var got Battery
	if err := json.Unmarshal(raw, &got); err != nil {
		return Battery{}, fmt.Errorf("%w: battery", driver.ErrUnavailable)
	}
	return got, nil
}

// ReadMode reads "light" or "dark" from the host appearance file.
func ReadMode(ctx context.Context) (string, error) {
	raw, err := readNamed(ctx, "daynight.txt")
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(string(raw)) {
	case "dark":
		return "dark", nil
	case "light":
		return "light", nil
	default:
		return "", fmt.Errorf("%w: daynight", driver.ErrUnavailable)
	}
}

func readNamed(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dir := dirFrom(ctx)
	if dir == "" {
		return nil, fmt.Errorf("%w: no ios host", driver.ErrUnavailable)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, name)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, name)
	}
	return raw, nil
}
