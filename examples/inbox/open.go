package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/driver/dirs"
)

var (
	errNilHandler  = errors.New("nil handler")
	errEmptyURL    = errors.New("empty url")
	errEmptyPaths  = errors.New("empty paths")
	errUnknownKind = errors.New("unknown kind")
	errOpenLine    = errors.New("open line")
)

// Hosts append one JSON object per line to Cache/open.jsonl.
// Listen also reads argv and ELETROCROMO_OPEN.

const openFileName = "open.jsonl"

const envOpen = "ELETROCROMO_OPEN"

type Kind int

const (
	KindURL Kind = iota + 1
	KindFiles
)

type Event struct {
	Kind  Kind
	URL   string
	Paths []string
}

type wire struct {
	Kind  string   `json:"kind"`
	URL   string   `json:"url,omitempty"`
	Paths []string `json:"paths,omitempty"`
}

func openFilePath(cacheDir string) string {
	return filepath.Join(cacheDir, openFileName)
}

func Listen(ctx context.Context, appID string, handle func(Event) error) error {
	if handle == nil {
		return errNilHandler
	}
	d, err := dirs.Resolve(ctx, appID)
	if err != nil {
		return err
	}
	for _, ev := range Collect(os.Args[1:], os.Getenv(envOpen)) {
		if err := handle(ev); err != nil {
			return err
		}
	}
	return TailFile(ctx, openFilePath(d.Cache), handle)
}

func Collect(args []string, env string) []Event {
	var out []Event
	for _, tok := range tokens(args, env) {
		if ev, ok := Token(tok); ok {
			out = append(out, ev)
		}
	}
	return out
}

func tokens(args []string, env string) []string {
	var out []string
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		out = append(out, a)
	}
	for a := range strings.FieldsSeq(env) {
		out = append(out, a)
	}
	return out
}

func Token(s string) (Event, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Event{}, false
	}
	if looksLikeURL(s) {
		return Event{Kind: KindURL, URL: s}, true
	}
	if filepath.IsAbs(s) || fileLooksLikePath(s) {
		return Event{Kind: KindFiles, Paths: []string{s}}, true
	}
	return Event{}, false
}

func looksLikeURL(s string) bool {
	if len(s) >= 3 && s[1] == ':' && (s[2] == '\\' || s[2] == '/') {
		return false
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" {
		return false
	}
	r := rune(u.Scheme[0])
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func fileLooksLikePath(s string) bool {
	if _, err := os.Stat(s); err == nil {
		return true
	}
	return strings.ContainsAny(s, `/\`)
}

func ParseLine(line []byte) (Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return Event{}, nil
	}
	var w wire
	if err := json.Unmarshal(line, &w); err != nil {
		return Event{}, fmt.Errorf("%w: %w", errOpenLine, err)
	}
	switch strings.ToLower(strings.TrimSpace(w.Kind)) {
	case "url":
		if strings.TrimSpace(w.URL) == "" {
			return Event{}, errEmptyURL
		}
		return Event{Kind: KindURL, URL: w.URL}, nil
	case "files":
		if len(w.Paths) == 0 {
			return Event{}, errEmptyPaths
		}
		return Event{Kind: KindFiles, Paths: append([]string(nil), w.Paths...)}, nil
	default:
		return Event{}, fmt.Errorf("%w: %q", errUnknownKind, w.Kind)
	}
}

func TailFile(ctx context.Context, path string, handle func(Event) error) error {
	var offset int64
	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()
	for {
		n, err := consume(path, offset, handle)
		if err != nil {
			return err
		}
		offset = n
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

func consume(path string, offset int64, handle func(Event) error) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return offset, nil
		}
		return offset, err
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), 1024*1024)
	for sc.Scan() {
		ev, err := ParseLine(sc.Bytes())
		if err != nil {
			return offset, err
		}
		if ev.Kind == 0 {
			offset += int64(len(sc.Bytes()) + 1)
			continue
		}
		if err := handle(ev); err != nil {
			return offset, err
		}
		offset += int64(len(sc.Bytes()) + 1)
	}
	return offset, sc.Err()
}
