package gotool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// RunSource compiles and runs a package main given as source.
// On Linux the source, the archive, and the binary are memfds. On
// Darwin and Windows they are temporary files. The directory used to
// find the runtime archives is removed before return, and it never
// contains a .go file.
func RunSource(ctx context.Context, src string) (string, error) {
	dir, err := os.MkdirTemp("", "lewkit-jit-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	compileCfg, linkDeps, err := runtimeDeps(dir)
	if err != nil {
		return "", err
	}
	stored, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return "", err
	}
	if len(stored) != 0 {
		return "", fmt.Errorf("%w: %v", ErrSourceWasStored, stored)
	}

	srcFile, err := openMem("src")
	if err != nil {
		return "", err
	}
	defer srcFile.Close()
	if _, err := srcFile.Write([]byte(src)); err != nil {
		return "", err
	}
	arcFile, err := openMem("archive")
	if err != nil {
		return "", err
	}
	defer arcFile.Close()
	outFile, err := openMem("bin")
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	srcPath := srcFile.Path()
	arcPath := arcFile.Path()
	outPath := outFile.Path()
	if err := Compile(ctx, "main", arcPath, compileCfg, "", "go1.27", false, []string{srcPath}); err != nil {
		return "", err
	}
	linkCfg := filepath.Join(dir, "link.importcfg")
	body := "# import config\npackagefile main=" + arcPath + "\n" + linkDeps
	if err := os.WriteFile(linkCfg, []byte(body), 0o644); err != nil {
		return "", err
	}
	if err := Link(ctx, outPath, linkCfg, "exe", arcPath); err != nil {
		return "", err
	}
	cmd := commandBuilt(outPath, outFile.File())
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w%w\n%s", ErrRunSource, err, stdout.Bytes())
	}
	return stdout.String(), nil
}
