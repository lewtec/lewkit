//go:build ignore

// genbundle pre-bundles the assembled CF worker for //go:embed.
//
//	mise run assemble && go run ./genbundle.go
package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lucasew/orvalho/pkg/workers/bundle"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	workerDir := filepath.Join(root, "worker")
	entry := filepath.Join(workerDir, "entry.mjs")
	if _, err := os.Stat(entry); err != nil {
		fatal(fmt.Errorf("%w: %s — run: mise run assemble", os.ErrNotExist, entry))
	}
	esbuild, err := exec.LookPath("esbuild")
	if err != nil {
		fatal(fmt.Errorf("esbuild not on PATH: %w", err))
	}

	script, err := bundle.BundleEntry(bundle.BundleOptions{
		PackageDir: workerDir,
		Entry:      "entry.mjs",
		Esbuild:    esbuild,
	})
	if err != nil {
		fatal(err)
	}

	embedDir := filepath.Join(root, "embed")
	assetsOut := filepath.Join(embedDir, "assets")
	if err := os.MkdirAll(assetsOut, 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(embedDir, "guest.js"), []byte(script), 0o644); err != nil {
		fatal(err)
	}

	// Refresh assets from worker/assets (client build).
	if err := os.RemoveAll(assetsOut); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(assetsOut, 0o755); err != nil {
		fatal(err)
	}
	srcAssets := filepath.Join(workerDir, "assets")
	if st, err := os.Stat(srcAssets); err == nil && st.IsDir() {
		if err := copyTree(srcAssets, assetsOut); err != nil {
			fatal(err)
		}
	}
	// Ensure embed has at least one assets file for go:embed.
	if empty, err := dirEmpty(assetsOut); err != nil {
		fatal(err)
	} else if empty {
		if err := os.WriteFile(filepath.Join(assetsOut, ".gitkeep"), nil, 0o644); err != nil {
			fatal(err)
		}
	}

	fmt.Printf("wrote embed/guest.js (%d bytes) and embed/assets/\n", len(script))
}

func dirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "genbundle: %v\n", err)
	os.Exit(1)
}
