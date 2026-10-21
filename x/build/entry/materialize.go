package entry

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

var mains sync.Map

var (
	rootOnce sync.Once
	rootDir  string
	rootErr  error
)

func materialize(ctx context.Context, parsed Entry, base, title string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	ref, err := resolveRef(parsed.Ref, base)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(ref)
	if err != nil {
		if os.IsNotExist(err) {
			return "", missing(parsed.Provider, ref)
		}
		return "", err
	}
	if parsed.Provider == ProviderAstro && !info.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrAstroConfig, ref)
	}
	print, err := fingerprint(ctx, parsed.Provider, ref, info)
	if err != nil {
		return "", err
	}
	key := string(parsed.Provider) + "\x00" + ref + "\x00" + displayTitle(title) + "\x00" + print
	if got, ok := mains.Load(key); ok {
		return got.(string), nil
	}
	root, err := lewkitRoot()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(root, ".lewkit-entry-")
	if err != nil {
		return "", err
	}
	switch parsed.Provider {
	case ProviderStatic:
		err = writeStatic(ctx, ref, info, dir, title)
	case ProviderAstro:
		err = writeAstro(ctx, ref, dir, title)
	default:
		err = fmt.Errorf("%w: %s", ErrEntryProvider, parsed.Provider)
	}
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	if existing, loaded := mains.LoadOrStore(key, dir); loaded {
		os.RemoveAll(dir)
		return existing.(string), nil
	}
	return dir, nil
}

func missing(provider Provider, ref string) error {
	if provider == ProviderAstro {
		return fmt.Errorf("%w: %s", ErrAstroConfig, ref)
	}
	return fmt.Errorf("%w: %s", ErrStaticMissing, ref)
}

func resolveRef(ref, base string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("%w", ErrEntrySyntax)
	}
	if !filepath.IsAbs(ref) {
		ref = filepath.Join(base, ref)
	}
	return filepath.Abs(ref)
}

func fingerprint(ctx context.Context, provider Provider, ref string, info os.FileInfo) (string, error) {
	if provider == ProviderAstro {
		return astroFingerprint(ref, info)
	}
	if !info.IsDir() {
		return fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano()), nil
	}
	var buf strings.Builder
	err := filepath.WalkDir(ref, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(ref, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&buf, "%s %d %d\n", rel, fileInfo.Size(), fileInfo.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeStatic(ctx context.Context, ref string, info os.FileInfo, dir, title string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	if info.IsDir() {
		if err := common.CopyDir(ref, filepath.Join(dir, "site")); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "main.go"), []byte(staticDirMain(title)), 0o644)
	}
	if err := common.CopyFile(ref, filepath.Join(dir, "site.zip"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(staticZipMain(title)), 0o644)
}

func lewkitRoot() (string, error) {
	rootOnce.Do(func() {
		rootDir, rootErr = findLewkitRoot()
	})
	return rootDir, rootErr
}

func findLewkitRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", ErrLewkitSource
	}
	dir := filepath.Dir(file)
	for {
		body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && moduleLewkit(body) {
			if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr != nil {
				return "", fmt.Errorf("%w: %s", ErrLewkitSource, dir)
			}
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrLewkitSource
		}
		dir = parent
	}
}

func moduleLewkit(body []byte) bool {
	for _, line := range bytes.Split(body, []byte("\n")) {
		if bytes.Equal(bytes.TrimSpace(line), []byte("module github.com/lewtec/lewkit")) {
			return true
		}
	}
	return false
}
