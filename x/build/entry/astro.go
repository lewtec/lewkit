package entry

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lucasew/orvalho/pkg/workers/bundle"
)

const maxBuildLog = 4096

func writeAstro(ctx context.Context, project, dir, title string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	if err := requireAstroConfig(project); err != nil {
		return err
	}
	if err := runAstroBuild(ctx, project); err != nil {
		return err
	}
	worker, err := os.MkdirTemp("", "lewkit-astro-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(worker)
	if err := assembleAstro(project, worker); err != nil {
		return err
	}
	esbuild, err := findEsbuild(ctx, project)
	if err != nil {
		return err
	}
	script, err := bundle.BundleEntry(bundle.BundleOptions{
		PackageDir: worker,
		Entry:      "entry.mjs",
		Esbuild:    esbuild,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAstroBuild, err)
	}
	embedDir := filepath.Join(dir, "embed")
	if err := os.MkdirAll(filepath.Join(embedDir, "assets"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(embedDir, "guest.js"), []byte(script), 0o644); err != nil {
		return err
	}
	if err := copyAstroAssets(worker, embedDir); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(astroMain(title)), 0o644)
}

func requireAstroConfig(project string) error {
	for _, name := range []string{"astro.config.mjs", "astro.config.ts", "astro.config.js"} {
		if fileExists(filepath.Join(project, name)) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrAstroConfig, project)
}

func runAstroBuild(ctx context.Context, project string) error {
	name, args, err := astroCommand(ctx, project)
	if err != nil {
		return err
	}
	cmd := execdriver.MustCommand(ctx, name, args...)
	cmd.Dir = project
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if err := execdriver.Run(ctx, cmd); err != nil {
		return buildFailed(log.Bytes(), err)
	}
	return nil
}

func buildFailed(log []byte, err error) error {
	text := strings.TrimSpace(string(log))
	if len(text) > maxBuildLog {
		text = text[len(text)-maxBuildLog:]
	}
	if text == "" {
		return fmt.Errorf("%w: %w", ErrAstroBuild, err)
	}
	return fmt.Errorf("%w: %s: %w", ErrAstroBuild, text, err)
}

func astroCommand(ctx context.Context, dir string) (string, []string, error) {
	if name, args, ok := astroLock(dir); ok {
		return name, args, nil
	}
	if _, err := execdriver.Which(ctx, "bun"); err == nil {
		return "bun", []string{"run", "build"}, nil
	}
	if _, err := execdriver.Which(ctx, "npm"); err != nil {
		return "", nil, fmt.Errorf("%w: %w", ErrAstroBuild, err)
	}
	return "npm", []string{"run", "build"}, nil
}

func astroLock(dir string) (string, []string, bool) {
	if fileExists(filepath.Join(dir, "bun.lock")) || fileExists(filepath.Join(dir, "bun.lockb")) {
		return "bun", []string{"run", "build"}, true
	}
	if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
		return "pnpm", []string{"run", "build"}, true
	}
	if fileExists(filepath.Join(dir, "yarn.lock")) {
		return "yarn", []string{"build"}, true
	}
	if fileExists(filepath.Join(dir, "package-lock.json")) {
		return "npm", []string{"run", "build"}, true
	}
	return "", nil, false
}

func assembleAstro(project, worker string) error {
	if err := os.MkdirAll(filepath.Join(worker, "assets"), 0o755); err != nil {
		return err
	}
	server := filepath.Join(project, "dist", "server")
	client := filepath.Join(project, "dist", "client")
	legacy := filepath.Join(project, "dist", "_worker.js")
	switch {
	case dirExists(server):
		if err := common.CopyDir(server, worker); err != nil {
			return err
		}
		if dirExists(client) {
			if err := common.CopyDir(client, filepath.Join(worker, "assets")); err != nil {
				return err
			}
		}
	case dirExists(legacy):
		if err := common.CopyDir(legacy, worker); err != nil {
			return err
		}
		if err := copyLegacyAssets(filepath.Join(project, "dist"), filepath.Join(worker, "assets")); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: %s", ErrAstroDist, filepath.Join(project, "dist"))
	}
	entryFile := filepath.Join(worker, "entry.mjs")
	indexFile := filepath.Join(worker, "index.js")
	if !fileExists(entryFile) && fileExists(indexFile) {
		if err := common.CopyFile(indexFile, entryFile, 0o644); err != nil {
			return err
		}
	}
	if !fileExists(entryFile) {
		return fmt.Errorf("%w: %s", ErrAstroDist, entryFile)
	}
	return nil
}

func copyLegacyAssets(dist, assets string) error {
	entries, err := os.ReadDir(dist)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Name() {
		case "_worker.js", "_routes.json", "wrangler.json", "wrangler.toml":
			continue
		}
		src := filepath.Join(dist, entry.Name())
		dst := filepath.Join(assets, entry.Name())
		if entry.IsDir() {
			if err := common.CopyDir(src, dst); err != nil {
				return err
			}
			continue
		}
		if err := common.CopyFile(src, dst, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func copyAstroAssets(worker, embedDir string) error {
	src := filepath.Join(worker, "assets")
	dst := filepath.Join(embedDir, "assets")
	if dirExists(src) {
		if err := common.CopyDir(src, dst); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return os.WriteFile(filepath.Join(dst, ".gitkeep"), nil, 0o644)
	}
	return nil
}

func findEsbuild(ctx context.Context, project string) (string, error) {
	candidates := []string{
		filepath.Join(project, "node_modules", ".bin", "esbuild"),
		filepath.Join(project, "node_modules", "esbuild", "bin", "esbuild"),
	}
	if path := orvalhoEsbuild(project); path != "" {
		candidates = append(candidates, path)
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	path, err := execdriver.Which(ctx, "esbuild")
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEsbuild, err)
	}
	return path, nil
}

func orvalhoEsbuild(project string) string {
	root := filepath.Join(project, "node_modules", ".orvalho")
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "esbuild@") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	dir := filepath.Join(root, names[len(names)-1])
	for _, rel := range []string{"bin/esbuild", "esbuild"} {
		path := filepath.Join(dir, rel)
		if fileExists(path) {
			return path
		}
	}
	return ""
}

func astroFingerprint(dir string, info os.FileInfo) (string, error) {
	var buf strings.Builder
	fmt.Fprintf(&buf, "dir %d\n", info.ModTime().UnixNano())
	names := []string{
		"package.json",
		"astro.config.mjs",
		"astro.config.ts",
		"astro.config.js",
		"bun.lock",
		"bun.lockb",
		"pnpm-lock.yaml",
		"yarn.lock",
		"package-lock.json",
	}
	for _, name := range names {
		fileInfo, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		fmt.Fprintf(&buf, "%s %d %d\n", name, fileInfo.Size(), fileInfo.ModTime().UnixNano())
	}
	return buf.String(), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
