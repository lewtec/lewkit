// Package androiddex writes the Android host classes.dex.
//
// The working directory is the apk package (x/build/gen/apk). It holds
// resids.go, the lewkit Java sources, and the javagen templates. Run
// compiles package lewkit, the resource class from those ids, and the
// androidx libraries that bytecode calls, then writes classes.dex there.
//
// javac and d8 are the compilers. They come from a JDK and an Android SDK
// that are already on the machine. Run does not install them. release
// build embeds the file and does not call this package.
//
//	//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate androiddex
package androiddex

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/generate"
)

const dexName = "classes.dex"

// Run writes classes.dex in dir.
// dir is the Android host package. An empty dir is [generate.ErrDirRequired].
func Run(ctx context.Context, dir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if dir == "" {
		return generate.ErrDirRequired
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	sources, err := hostSources(ctx, dir)
	if err != nil {
		return err
	}
	javac, javaBin, err := javaTools(ctx)
	if err != nil {
		return err
	}
	androidJar, d8Jar, err := androidPlatform(ctx)
	if err != nil {
		return err
	}
	cache, err := cacheDir()
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lewkit-androiddex-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	libs, err := fetchLibraries(ctx, cache, work)
	if err != nil {
		return err
	}
	for rel, body := range libs.sources {
		sources[rel] = body
	}
	classDir := filepath.Join(work, "classes")
	if err := os.Mkdir(classDir, 0o755); err != nil {
		return err
	}
	if err := compileJava(ctx, javac, javaInputs{
		work:       work,
		classDir:   classDir,
		androidJar: androidJar,
		classpath:  libs.classpath,
		sources:    sources,
	}); err != nil {
		return err
	}
	program := filepath.Join(work, "program.jar")
	if err := jarClassTree(ctx, classDir, program); err != nil {
		return err
	}
	outDir := filepath.Join(work, "dex")
	if err := os.Mkdir(outDir, 0o755); err != nil {
		return err
	}
	if err := runD8(ctx, javaBin, d8Inputs{
		d8Jar:      d8Jar,
		androidJar: androidJar,
		outDir:     outDir,
		inputs:     append([]string{program}, libs.classpath...),
	}); err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(outDir, dexName))
	if err != nil {
		return err
	}
	if err := checkDex(raw); err != nil {
		return err
	}
	slog.InfoContext(ctx, "android dex", "dir", dir, "bytes", len(raw))
	return generate.WriteFile(filepath.Join(dir, dexName), raw)
}
