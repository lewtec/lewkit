package media

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lewtec/lewkit/x/driver/notification"
)

// RunAction runs next, previous, play-pause, stop, or show, then posts the track.
// next, previous, and play-pause wait briefly so the player can publish metadata.
func RunAction(ctx context.Context, action string) error {
	var err error
	switch action {
	case "next":
		err = Next(ctx)
	case "previous":
		err = Previous(ctx)
	case "play-pause":
		err = PlayPause(ctx)
	case "stop":
		err = Stop(ctx)
	case "show":
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	if err != nil {
		return err
	}
	if action == "next" || action == "previous" || action == "play-pause" {
		time.Sleep(200 * time.Millisecond)
	}
	return ShowStatus(ctx)
}

// ShowStatus reads the current track and posts it.
func ShowStatus(ctx context.Context) error {
	meta, err := GetMetadata(ctx)
	if err != nil {
		return err
	}
	return Notify(ctx, meta)
}

// Notify posts meta. An empty title is skipped.
func Notify(ctx context.Context, meta *Metadata) error {
	note, ok := statusNote(meta)
	if !ok {
		return nil
	}
	if meta.ArtUrl != "" {
		icon, err := GetArtCachePath(ctx, meta.ArtUrl)
		if err != nil {
			slog.ErrorContext(ctx, "media art", "error", err)
		} else {
			note.Icon = icon
		}
	}
	return notification.Notify(ctx, note)
}

// WatchStatus blocks until ctx is done and posts each metadata change.
func WatchStatus(ctx context.Context) error {
	return Watch(ctx, func(meta *Metadata) {
		if err := Notify(ctx, meta); err != nil {
			slog.ErrorContext(ctx, "media status", "error", err)
		}
	})
}

func statusNote(meta *Metadata) (notification.Notification, bool) {
	if meta == nil || meta.Title == "" {
		return notification.Notification{}, false
	}
	progress := 0.0
	if meta.Length > 0 {
		progress = float64(meta.Position) / float64(meta.Length)
	}
	title := meta.Title
	message := meta.Artist
	if message == "" {
		message = "Unknown Artist"
	}
	return notification.Notification{
		ID:          notification.StatusID,
		Title:       title,
		Message:     message,
		Progress:    progress,
		HasProgress: true,
	}, true
}
