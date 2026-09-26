// Package media controls an MPRIS player.
//
//	err = media.PlayPause(ctx)
//	meta, err := media.GetMetadata(ctx)
//	note, ok := media.StatusNotification(meta)
//
// Import [github.com/lewtec/lewkit/x/driver/media/dbus]. Album art from
// http(s) is cached under os.UserCacheDir()/lewkit/media-art.
package media

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

// ErrNoPlayer means no MPRIS player is on the session bus.
var ErrNoPlayer = errors.New("no media player")

// PlaybackStatus is the MPRIS playback state.
type PlaybackStatus string

const (
	StatusPlaying PlaybackStatus = "Playing"
	StatusPaused  PlaybackStatus = "Paused"
	StatusStopped PlaybackStatus = "Stopped"
)

// Metadata is the current track.
type Metadata struct {
	Title    string
	Artist   string
	ArtUrl   string
	Length   int64 // microseconds
	Position int64 // microseconds
	Status   PlaybackStatus
	Player   string
}

// Driver controls one MPRIS player.
type Driver interface {
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	PlayPause(ctx context.Context) error
	Stop(ctx context.Context) error
	GetMetadata(ctx context.Context) (*Metadata, error)
	// Watch blocks and calls callback when metadata changes.
	Watch(ctx context.Context, callback func(*Metadata)) error
}

// Next skips to the next track.
func Next(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Next(ctx) })
}

// Previous skips to the previous track.
func Previous(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Previous(ctx) })
}

// PlayPause toggles playback.
func PlayPause(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.PlayPause(ctx) })
}

// Stop stops playback.
func Stop(ctx context.Context) error {
	return driver.With(ctx, func(d Driver) error { return d.Stop(ctx) })
}

// GetMetadata returns the best player's current track.
func GetMetadata(ctx context.Context) (*Metadata, error) {
	return driver.WithResult(ctx, func(d Driver) (*Metadata, error) { return d.GetMetadata(ctx) })
}

// Watch blocks until ctx is done and calls callback when metadata changes.
func Watch(ctx context.Context, callback func(*Metadata)) error {
	return driver.With(ctx, func(d Driver) error { return d.Watch(ctx, callback) })
}

// GetArtCachePath returns a local path for url.
// file:// is returned as-is. Other non-http values are returned unchanged.
// http(s) bodies are stored under os.UserCacheDir()/lewkit/media-art.
func GetArtCachePath(ctx context.Context, url string) (string, error) {
	if after, ok := strings.CutPrefix(url, "file://"); ok {
		return after, nil
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return url, nil
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(cacheRoot, "lewkit", "media-art")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	hash := fmt.Sprintf("%x", md5.Sum([]byte(url)))
	path := filepath.Join(cacheDir, hash)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	if err := writeAtomic(path, resp.Body); err != nil {
		return "", err
	}
	return path, nil
}

func writeAtomic(path string, r io.Reader) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".part-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmp)
		}
	}()
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	return nil
}
