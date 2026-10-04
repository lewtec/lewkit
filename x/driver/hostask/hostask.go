// Package hostask asks the packaged host to present a dialog.
//
// The host watches ELETROCROMO_ASK_DIR, or <cache>/ask in app mode.
// Go writes request.json and waits for reply.txt. The reply is an
// askwire message: a status line and an optional payload.
//
// Kinds are alert 1, confirm 2, prompt 3, and choose 4.
// A test can install a hook with Set.
package hostask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/driver"
)

const (
	// KindAlert shows a message and waits for OK.
	KindAlert = 1
	// KindConfirm asks yes or no.
	KindConfirm = 2
	// KindPrompt reads one line.
	KindPrompt = 3
	// KindChoose picks one row. Body is one label per line.
	KindChoose = 4
)

// Fn presents one dialog and returns the askwire text.
type Fn func(ctx context.Context, kind int, title, body string) (string, error)

var hook sync.Mutex
var hooked Fn

// Set installs fn. Nil clears it. Tests and an embedded host use this.
func Set(fn Fn) {
	hook.Lock()
	hooked = fn
	hook.Unlock()
}

// Available reports whether Call can present a dialog.
func Available() bool {
	hook.Lock()
	ok := hooked != nil
	hook.Unlock()
	if ok {
		return true
	}
	return Dir() != ""
}

// Dir is the directory the host watches.
// ELETROCROMO_ASK_DIR wins. In app mode, <ELETROCROMO_CACHE_DIR>/ask is the fallback.
func Dir() string {
	if dir := os.Getenv("ELETROCROMO_ASK_DIR"); dir != "" {
		return dir
	}
	if !driver.AppMode() {
		return ""
	}
	cache := os.Getenv("ELETROCROMO_CACHE_DIR")
	if cache == "" {
		cache = os.Getenv("LEWKIT_CACHE_DIR")
	}
	if cache == "" {
		return ""
	}
	return filepath.Join(cache, "ask")
}

type request struct {
	Kind  int    `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Style string `json:"style,omitempty"`
}

var fileMu sync.Mutex

// Call presents one dialog. A hook wins over the directory.
func Call(ctx context.Context, kind int, title, body string) (string, error) {
	return CallStyled(ctx, kind, title, body, "")
}

// CallStyled is [Call] with a message-box style. The hook does not see style.
// The ask file records it for the host.
func CallStyled(ctx context.Context, kind int, title, body, style string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	hook.Lock()
	fn := hooked
	hook.Unlock()
	if fn != nil {
		return fn(ctx, kind, title, body)
	}
	dir := Dir()
	if dir == "" {
		return "", fmt.Errorf("%w: no ask host", driver.ErrUnavailable)
	}
	return fileCall(ctx, dir, kind, title, body, style)
}

func fileCall(ctx context.Context, dir string, kind int, title, body, style string) (string, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	reqPath := filepath.Join(dir, "request.json")
	replyPath := filepath.Join(dir, "reply.txt")
	_ = os.Remove(replyPath)
	payload, err := json.Marshal(request{Kind: kind, Title: title, Body: body, Style: style})
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
