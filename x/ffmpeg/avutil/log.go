package avutil

import (
	"fmt"
	"io"
	"sync"
)

// Logger writes FFmpeg-style lines to one writer.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  Level
	prefix string
}

// NewLogger writes lines at level and above. A nil writer discards them.
func NewLogger(out io.Writer, level Level) *Logger {
	if out == nil {
		out = io.Discard
	}
	return &Logger{out: out, level: level}
}

// SetLevel changes the threshold.
func (l *Logger) SetLevel(level Level) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

// Level is the threshold.
func (l *Logger) Level() Level {
	if l == nil {
		return LevelInfo
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// WithPrefix returns a logger that tags lines with prefix.
func (l *Logger) WithPrefix(prefix string) *Logger {
	if l == nil {
		return NewLogger(nil, LevelInfo)
	}
	return &Logger{out: l.out, level: l.level, prefix: prefix}
}

// Print writes text when level is at least the threshold.
func (l *Logger) Print(level Level, format string, args ...any) {
	if l == nil {
		return
	}
	text := fmt.Sprintf(format, args...)
	l.mu.Lock()
	defer l.mu.Unlock()
	if level > l.level {
		return
	}
	if l.prefix != "" {
		fmt.Fprintf(l.out, "[%s] %s\n", l.prefix, text)
		return
	}
	fmt.Fprintln(l.out, text)
}
