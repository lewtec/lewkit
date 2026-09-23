// Package logging writes slog lines as a level letter, the message, then
// key=value pairs. A terminal colors the level letter and dims the keys.
// NO_COLOR, TERM=dumb, CI, and a non-tty stderr stay plain.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// NewHandler returns a handler that writes compact log lines to w.
// Color follows the process stderr: on for a terminal, off when NO_COLOR
// is set, TERM is dumb, CI is set, or stderr is not a terminal.
func NewHandler(w io.Writer, opts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &Handler{
		w:        w,
		opts:     *opts,
		useColor: colorEnabled(),
	}
}

// Handler is a slog.Handler for [NewHandler].
type Handler struct {
	w        io.Writer
	opts     slog.HandlerOptions
	pre      []slog.Attr
	groups   []string
	useColor bool
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.opts.Level != nil {
		min = h.opts.Level.Level()
	}
	return level >= min
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}
	attrs := make([]slog.Attr, 0, len(h.pre)+r.NumAttrs())
	attrs = append(attrs, h.pre...)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, h.prefix(a))
		return true
	})
	attrs = replaceAttrs(h.opts.ReplaceAttr, attrs)

	var line string
	if h.useColor {
		line = formatColored(r.Level, r.Message, attrs)
	} else {
		line = formatPlain(r.Level, r.Message, attrs)
	}
	if h.w == nil {
		return nil
	}
	_, err := fmt.Fprintln(h.w, line)
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	nh := h.clone()
	nh.pre = make([]slog.Attr, len(h.pre), len(h.pre)+len(attrs))
	copy(nh.pre, h.pre)
	for _, a := range attrs {
		nh.pre = append(nh.pre, h.prefix(a))
	}
	return nh
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	nh := h.clone()
	nh.groups = make([]string, len(h.groups)+1)
	copy(nh.groups, h.groups)
	nh.groups[len(h.groups)] = name
	return nh
}

func (h *Handler) clone() *Handler {
	return &Handler{
		w:        h.w,
		opts:     h.opts,
		pre:      h.pre,
		groups:   h.groups,
		useColor: h.useColor,
	}
}

func (h *Handler) prefix(a slog.Attr) slog.Attr {
	if len(h.groups) == 0 || a.Key == "" {
		return a
	}
	a.Key = strings.Join(h.groups, ".") + "." + a.Key
	return a
}

func replaceAttrs(replace func([]string, slog.Attr) slog.Attr, attrs []slog.Attr) []slog.Attr {
	if replace == nil {
		return attrs
	}
	out := attrs[:0]
	for _, a := range attrs {
		a = replace(nil, a)
		if a.Equal(slog.Attr{}) {
			continue
		}
		out = append(out, a)
	}
	return out
}

// colorEnabled reports whether lines should include ANSI color.
// When os.Stderr is not a terminal (the progress view may capture it),
// the check falls back to /dev/stderr so the original tty still counts.
func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || os.Getenv("CI") != "" {
		return false
	}
	if fi, err := os.Stderr.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		return true
	}
	f, err := os.OpenFile("/dev/stderr", os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func levelLetter(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "D"
	case level < slog.LevelWarn:
		return "I"
	case level < slog.LevelError:
		return "W"
	default:
		return "E"
	}
}

func coloredLevel(level slog.Level) string {
	letter := levelLetter(level)
	switch {
	case level < slog.LevelInfo:
		return "\x1b[100m" + letter + "\x1b[0m"
	case level < slog.LevelWarn:
		return "\x1b[46;1m" + letter + "\x1b[0m"
	case level < slog.LevelError:
		return "\x1b[43;1m" + letter + "\x1b[0m"
	default:
		return "\x1b[41;1m" + letter + "\x1b[0m"
	}
}

func formatPlain(level slog.Level, msg string, attrs []slog.Attr) string {
	var b strings.Builder
	b.WriteString(levelLetter(level))
	if msg != "" {
		b.WriteByte(' ')
		b.WriteString(msg)
	}
	for _, a := range attrs {
		appendLogAttr(&b, a, false)
	}
	return b.String()
}

func formatColored(level slog.Level, msg string, attrs []slog.Attr) string {
	var b strings.Builder
	b.WriteString(coloredLevel(level))
	if msg != "" {
		b.WriteByte(' ')
		b.WriteString(msg)
	}
	for _, a := range attrs {
		appendLogAttr(&b, a, true)
	}
	return b.String()
}

func writePlainKV(b *strings.Builder, key string, v slog.Value) {
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(formatValue(v))
}

func writeColoredKV(b *strings.Builder, key string, v slog.Value) {
	b.WriteString(" \x1b[2;36m")
	b.WriteString(key)
	b.WriteString("\x1b[0m\x1b[2m=\x1b[0m")
	b.WriteString(formatValue(v))
}

func appendLogAttr(b *strings.Builder, a slog.Attr, color bool) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() != slog.KindGroup {
		if color {
			writeColoredKV(b, a.Key, a.Value)
		} else {
			writePlainKV(b, a.Key, a.Value)
		}
		return
	}
	for _, sub := range a.Value.Group() {
		key := sub.Key
		if a.Key != "" {
			key = a.Key + "." + sub.Key
		}
		appendLogAttr(b, slog.Attr{Key: key, Value: sub.Value}, color)
	}
}

func formatValue(v slog.Value) string {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return quote(v.String())
	case slog.KindBool:
		return strconv.FormatBool(v.Bool())
	case slog.KindInt64:
		return strconv.FormatInt(v.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(v.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(v.Float64(), 'g', -1, 64)
	case slog.KindGroup:
		parts := make([]string, 0, len(v.Group()))
		for _, sub := range v.Group() {
			parts = append(parts, sub.Key+"="+formatValue(sub.Value))
		}
		return "{" + strings.Join(parts, " ") + "}"
	default:
		return quote(fmt.Sprint(v.Any()))
	}
}

func quote(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\r\n=\"") {
		return strconv.Quote(s)
	}
	return s
}
