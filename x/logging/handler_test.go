package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatPlain(t *testing.T) {
	attrs := []slog.Attr{
		slog.String("path", "/tmp/x"),
		slog.String("note", "has space"),
		slog.Int("n", 3),
		slog.Group("req", slog.String("id", "a")),
	}
	got := formatPlain(slog.LevelInfo, "ready", attrs)
	assert.Equal(t, `I ready path=/tmp/x note="has space" n=3 req.id=a`, got)
	assert.Equal(t, "W disk", formatPlain(slog.LevelWarn, "disk", nil))
	assert.Equal(t, "E boom", formatPlain(slog.LevelError, "boom", nil))
	assert.Equal(t, "D trace", formatPlain(slog.LevelDebug-4, "trace", nil))
}

func TestFormatColored(t *testing.T) {
	got := formatColored(slog.LevelInfo, "up", []slog.Attr{slog.String("k", "v")})
	assert.Equal(t, "\x1b[46;1mI\x1b[0m up \x1b[2;36mk\x1b[0m\x1b[2m=\x1b[0mv", got)
	assert.Equal(t, "\x1b[100mD\x1b[0m", formatColored(slog.LevelDebug, "", nil))
	assert.Equal(t, "\x1b[100mD\x1b[0m deep", formatColored(slog.LevelDebug-4, "deep", nil))
	assert.Equal(t, "\x1b[43;1mW\x1b[0m", formatColored(slog.LevelWarn, "", nil))
	assert.Equal(t, "\x1b[41;1mE\x1b[0m", formatColored(slog.LevelError, "", nil))
}

func TestHandlerPlainLine(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	h := NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(h).With("op", "close").WithGroup("req")
	logger.Info("saved", "id", "a")
	assert.Equal(t, "I saved op=close req.id=a\n", buf.String())
}

func TestHandlerDropsBelowLevel(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	logger := slog.New(NewHandler(&buf, nil))
	logger.Debug("hidden")
	logger.Warn("shown")
	assert.Equal(t, "W shown\n", buf.String())
}

func TestHandlerReplaceAttrDrops(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	h := NewHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "secret" {
				return slog.Attr{}
			}
			return a
		},
	})
	slog.New(h).Info("open", "secret", "nope", "path", "/tmp")
	assert.Equal(t, "I open path=/tmp\n", buf.String())
}

func TestHandlerQuotesAndKinds(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var buf bytes.Buffer
	logger := slog.New(NewHandler(&buf, nil))
	logger.Error("bad", "empty", "", "ok", true, "f", 1.5, "note", "disk full")
	line := strings.TrimSuffix(buf.String(), "\n")
	assert.Equal(t, `E bad empty="" ok=true f=1.5 note="disk full"`, line)
}

func TestColorEnabledHonorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	assert.False(t, colorEnabled())
}

func TestHandlerNilWriter(t *testing.T) {
	h := NewHandler(nil, nil)
	require.NoError(t, h.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "x", 0)))
}
