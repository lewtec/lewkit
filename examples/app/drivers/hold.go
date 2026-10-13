package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/driver/window"
)

// held is process state for drivers whose API is open/close rather than a reading.
type held struct {
	mu       sync.Mutex
	win      window.Window
	web      webview.View
	webTitle string
	tray     tray.Tray
	trayTip  string
}

func (h *held) windowView() (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.win == nil {
		return false, ""
	}
	size := h.win.Size()
	return true, fmt.Sprintf("%dx%d", size.X, size.Y)
}

func (h *held) openWindow(ctx context.Context, cfg window.Config) (string, error) {
	win, err := window.Open(ctx, cfg)
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	busy := h.win != nil
	if !busy {
		h.win = win
	}
	h.mu.Unlock()
	if busy {
		if err := win.Close(); err != nil {
			slog.Error("window close", "err", err)
		}
		return "", errWindowOpen
	}
	size := win.Size()
	return fmt.Sprintf("%dx%d", size.X, size.Y), nil
}

func (h *held) closeWindow() error {
	h.mu.Lock()
	win := h.win
	h.win = nil
	h.mu.Unlock()
	if win == nil {
		return errNoWindow
	}
	return win.Close()
}

func (h *held) webView() (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.web == nil {
		return false, ""
	}
	return true, h.webTitle
}

func (h *held) openWeb(ctx context.Context, cfg webview.Config) error {
	view, err := webview.Open(ctx, cfg)
	if err != nil {
		return err
	}
	h.mu.Lock()
	busy := h.web != nil
	if !busy {
		h.web = view
		h.webTitle = cfg.Title
	}
	h.mu.Unlock()
	if busy {
		if err := view.Close(); err != nil {
			slog.Error("webview close", "err", err)
		}
		return errWebOpen
	}
	context.AfterFunc(ctx, func() {
		h.mu.Lock()
		current := h.web == view
		if current {
			h.web = nil
			h.webTitle = ""
		}
		h.mu.Unlock()
		if current {
			if err := view.Close(); err != nil {
				slog.Error("webview close", "err", err)
			}
		}
	})
	return nil
}

func (h *held) closeWeb() error {
	h.mu.Lock()
	view := h.web
	h.web = nil
	h.webTitle = ""
	h.mu.Unlock()
	if view == nil {
		return errNoWeb
	}
	return view.Close()
}

func (h *held) evalWeb(ctx context.Context, script string) (string, error) {
	h.mu.Lock()
	view := h.web
	h.mu.Unlock()
	if view == nil {
		return "", errNoWeb
	}
	return view.Evaluate(ctx, script)
}

func (h *held) trayView() (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tray == nil {
		return false, ""
	}
	return true, h.trayTip
}

func (h *held) openTray(ctx context.Context, cfg tray.Config) error {
	item, err := tray.Open(ctx, cfg)
	if err != nil {
		return err
	}
	h.mu.Lock()
	busy := h.tray != nil
	if !busy {
		h.tray = item
		h.trayTip = tray.Tip(cfg)
	}
	h.mu.Unlock()
	if busy {
		if err := item.Close(); err != nil {
			slog.Error("tray close", "err", err)
		}
		return errTrayOpen
	}
	context.AfterFunc(ctx, func() {
		h.mu.Lock()
		current := h.tray == item
		if current {
			h.tray = nil
			h.trayTip = ""
		}
		h.mu.Unlock()
		if current {
			if err := item.Close(); err != nil {
				slog.Error("tray close", "err", err)
			}
		}
	})
	return nil
}

func (h *held) updateTray(cfg tray.Config) error {
	h.mu.Lock()
	item := h.tray
	h.mu.Unlock()
	if item == nil {
		return errNoTray
	}
	if err := item.Update(cfg); err != nil {
		return err
	}
	h.mu.Lock()
	h.trayTip = tray.Tip(cfg)
	h.mu.Unlock()
	return nil
}

func (h *held) closeTray() error {
	h.mu.Lock()
	item := h.tray
	h.tray = nil
	h.trayTip = ""
	h.mu.Unlock()
	if item == nil {
		return errNoTray
	}
	return item.Close()
}
