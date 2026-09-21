// Package mise is the mise backend.
//
// A spec mise:node@22 installs that package with the mise binary on PATH
// and symlinks the named command under the store's bin directory.
package mise

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
)

var (
	// ErrEmptyMiseRef is returned when a mise ref is blank.
	ErrEmptyMiseRef = errors.New("mise ref cannot be empty")
	// ErrMissingMiseSpec is returned when an install spec is blank.
	ErrMissingMiseSpec = errors.New("missing mise artifact spec")
	// ErrMiseNotFound is returned when the mise binary is not on PATH.
	ErrMiseNotFound = errors.New("mise binary not found on PATH")
)

func init() {
	tool.Register("mise", &Backend{})
}

// Backend is the mise backend.
type Backend struct{}

// Name returns a short description of the backend.
func (backend *Backend) Name() string { return "mise" }

// Tool returns the tool for ref (for example "node" or "python").
func (backend *Backend) Tool(ref string) (tool.Tool, error) {
	return NewTool(ref)
}

// MiseTool installs one mise package.
type MiseTool struct {
	spec string
}

// NewTool constructs a MiseTool for ref.
func NewTool(ref string) (tool.Tool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrEmptyMiseRef
	}
	return &MiseTool{spec: ref}, nil
}

// ListVersions returns the single version from `mise latest`.
func (installed *MiseTool) ListVersions(ctx context.Context) ([]string, error) {
	version, err := output(ctx, "latest", installed.spec)
	if err != nil {
		return nil, err
	}
	if version == "" {
		return nil, fmt.Errorf("mise latest returned empty version for %q", installed.spec)
	}
	return []string{version}, nil
}

// Install runs mise install and symlinks the primary command into destination/bin.
func (installed *MiseTool) Install(ctx context.Context, version, destination string) error {
	toolSpec := installed.spec
	if !strings.Contains(toolSpec, "@") && version != "" && version != "latest" {
		toolSpec = toolSpec + "@" + strings.TrimSpace(version)
	}
	if err := run(ctx, "install", toolSpec); err != nil {
		return err
	}
	commandName := installed.spec
	if index := strings.IndexAny(commandName, "@:"); index > 0 {
		commandName = commandName[:index]
	}
	_, err := installed.EnsureBinary(ctx, version, commandName, destination)
	return err
}

// EnsureBinary installs the mise package when needed and symlinks commandName into destination/bin.
func (installed *MiseTool) EnsureBinary(ctx context.Context, version, commandName, destination string) (string, error) {
	toolSpec := strings.TrimSpace(installed.spec) + "@" + strings.TrimSpace(version)
	binaryPath, err := resolveBinary(ctx, commandName, toolSpec)
	if err != nil {
		if err := run(ctx, "install", toolSpec); err != nil {
			return "", err
		}
		binaryPath, err = resolveBinary(ctx, commandName, toolSpec)
		if err != nil {
			return "", err
		}
	}
	return symlinkBinary(destination, binaryPath, commandName)
}

func resolveBinary(ctx context.Context, binaryName, toolSpec string) (string, error) {
	root, err := output(ctx, "where", toolSpec)
	if err != nil {
		return "", err
	}
	if binaryPath := tool.FindBinary(root, binaryName); binaryPath != "" {
		return binaryPath, nil
	}
	return "", fmt.Errorf("%w: %q under %s", tool.ErrBinaryNotFound, binaryName, root)
}

func symlinkBinary(destination, binaryPath, commandName string) (string, error) {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return "", err
	}
	linkPath := filepath.Join(destination, "bin", commandName)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Lstat(linkPath); err == nil {
		if err := os.Remove(linkPath); err != nil {
			return "", err
		}
	}
	if err := os.Symlink(binaryPath, linkPath); err != nil {
		return "", err
	}
	return linkPath, nil
}

func miseBinary() (string, error) {
	path, err := exec.LookPath("mise")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMiseNotFound, err)
	}
	return path, nil
}

func output(ctx context.Context, args ...string) (string, error) {
	binary, err := miseBinary()
	if err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, binary, args...)
	out, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func run(ctx context.Context, args ...string) error {
	spec := ""
	if len(args) > 0 {
		spec = strings.TrimSpace(args[len(args)-1])
	}
	if spec == "" {
		return ErrMissingMiseSpec
	}
	binary, err := miseBinary()
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, binary, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	return command.Run()
}
