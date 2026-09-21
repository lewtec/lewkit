package tool

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrCheckFailed is returned when one or more install checks do not pass.
	ErrCheckFailed = errors.New("install check failed")
	// ErrBinaryNotFound is returned when a named binary is missing from an install directory.
	ErrBinaryNotFound = errors.New("binary not found")
	// ErrEmptyRelativePath is returned when a check path is empty or ".".
	ErrEmptyRelativePath = errors.New("empty relative path")
	// ErrPathEscapes is returned when a check path resolves outside the install directory.
	ErrPathEscapes = errors.New("path escapes install directory")
)

// Checks returns list unchanged. It keeps call sites that build a check slice readable.
func Checks(list ...Check) []Check {
	return list
}

// RunChecks runs InstallChecks when installed implements Checker.
// A tool that does not implement Checker, or that returns no checks, is a success.
func RunChecks(ctx context.Context, destination string, installed Tool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	checker, ok := installed.(Checker)
	if !ok {
		return nil
	}
	list := checker.InstallChecks()
	if len(list) == 0 {
		return nil
	}
	var errs []error
	for _, check := range list {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		if err := check.Check(ctx, destination); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", check.Name(), err))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrCheckFailed, errors.Join(errs...))
}

// FileExists checks that destination/relativePath exists as a regular file.
func FileExists(relativePath string) Check {
	return pathCheck{
		name:         "exists:" + relativePath,
		relativePath: relativePath,
		fn: func(_ context.Context, path string, info fs.FileInfo) error {
			if info.IsDir() {
				return fmt.Errorf("%s is a directory, want a file", path)
			}
			return nil
		},
	}
}

// Executable checks that destination/relativePath exists and is executable.
// On Windows, a non-directory file is enough.
func Executable(relativePath string) Check {
	return pathCheck{
		name:         "executable:" + relativePath,
		relativePath: relativePath,
		fn: func(_ context.Context, path string, info fs.FileInfo) error {
			if info.IsDir() {
				return fmt.Errorf("%s is a directory, want an executable file", path)
			}
			if runtime.GOOS == "windows" {
				return nil
			}
			if info.Mode()&0o111 == 0 {
				return fmt.Errorf("%s is not executable", path)
			}
			return nil
		},
	}
}

// Binary locates binaryName under destination, then requires that file to exist and be executable.
func Binary(binaryName string) Check {
	return binaryCheck{binaryName: binaryName}
}

type pathCheck struct {
	name         string
	relativePath string
	fn           func(ctx context.Context, path string, info fs.FileInfo) error
}

func (check pathCheck) Name() string { return check.name }

func (check pathCheck) Check(ctx context.Context, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := safeJoin(destination, check.relativePath)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: %w", path, fs.ErrNotExist)
		}
		return err
	}
	return check.fn(ctx, path, info)
}

type binaryCheck struct {
	binaryName string
}

func (check binaryCheck) Name() string { return "binary:" + check.binaryName }

func (check binaryCheck) Check(ctx context.Context, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := FindBinary(destination, check.binaryName)
	if path == "" {
		return fmt.Errorf("%w: %q in %s", ErrBinaryNotFound, check.binaryName, destination)
	}
	relative, err := filepath.Rel(destination, path)
	if err != nil {
		return err
	}
	if err := FileExists(relative).Check(ctx, destination); err != nil {
		return err
	}
	return Executable(relative).Check(ctx, destination)
}

func safeJoin(destination, relativePath string) (string, error) {
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	cleanRelative := filepath.Clean("/" + filepath.ToSlash(relativePath))
	cleanRelative = strings.TrimPrefix(cleanRelative, "/")
	if cleanRelative == "" || cleanRelative == "." {
		return "", ErrEmptyRelativePath
	}
	joined := filepath.Join(destinationAbs, filepath.FromSlash(cleanRelative))
	joinedAbs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	separator := string(os.PathSeparator)
	if joinedAbs != destinationAbs && !strings.HasPrefix(joinedAbs, destinationAbs+separator) {
		return "", fmt.Errorf("%w: %q", ErrPathEscapes, relativePath)
	}
	return joinedAbs, nil
}
