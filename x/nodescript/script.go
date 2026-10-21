// Package nodescript runs a package.json script.
// The lockfile selects bun, pnpm, or npm. Nothing else does.
// A missing manager is installed with x/tool.
// Install runs before the script. The child PATH ends with node_modules/.bin.
package nodescript

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/tool"
	_ "github.com/lewtec/lewkit/x/tool/registry/bun"
	_ "github.com/lewtec/lewkit/x/tool/registry/nodejs"
	_ "github.com/lewtec/lewkit/x/tool/registry/pnpm"
)

const maxLog = 4096

var (
	ErrLockfile = errs.Wrap(errs.ErrNotFound, "package lockfile")
	ErrScript   = errs.Wrap(errs.ErrInvalid, "package script")
	ErrInstall  = errs.Wrap(errs.ErrInvalid, "package install")
	ErrManager  = errs.Wrap(errs.ErrNotFound, "package manager")
)

// Manager is the package manager selected by one lockfile.
// Command is the program. Spec is the x/tool spec. Binary is the file name.
type Manager struct {
	Command string
	Spec    string
	Binary  string
}

type lockfile struct {
	name    string
	manager Manager
}

// bun wins over pnpm, and pnpm wins over npm, when more than one lockfile exists.
var lockfiles = []lockfile{
	{name: "bun.lock", manager: Manager{Command: "bun", Spec: "bun", Binary: "bun"}},
	{name: "bun.lockb", manager: Manager{Command: "bun", Spec: "bun", Binary: "bun"}},
	{name: "pnpm-lock.yaml", manager: Manager{Command: "pnpm", Spec: "pnpm", Binary: "pnpm"}},
	{name: "package-lock.json", manager: Manager{Command: "npm", Spec: "nodejs", Binary: "npm"}},
}

// For returns the manager for directory.
// The choice is the lockfile. PATH is not consulted.
func For(directory string) (Manager, error) {
	for _, item := range lockfiles {
		info, err := os.Stat(filepath.Join(directory, item.name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return Manager{}, err
		}
		if info.IsDir() {
			continue
		}
		return item.manager, nil
	}
	return Manager{}, fmt.Errorf("%w: %s", ErrLockfile, directory)
}

// Run installs dependencies in directory, then runs the named script.
func Run(ctx context.Context, directory, script string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	script = strings.TrimSpace(script)
	if script == "" {
		return fmt.Errorf("%w", ErrScript)
	}
	manager, err := For(directory)
	if err != nil {
		return err
	}
	binary, err := resolveBinary(ctx, manager)
	if err != nil {
		return err
	}
	install := invocation{directory: directory, binary: binary, sentinel: ErrInstall}
	if err := install.run(ctx, "install"); err != nil {
		return err
	}
	scriptRun := invocation{directory: directory, binary: binary, sentinel: ErrScript}
	return scriptRun.run(ctx, "run", script)
}

// resolveBinary is replaced by tests.
var resolveBinary = resolveOnHost

func resolveOnHost(ctx context.Context, manager Manager) (string, error) {
	found, err := execdriver.Which(ctx, manager.Binary)
	if err == nil {
		return found, nil
	}
	if !errors.Is(err, execdriver.ErrNotFound) {
		return "", err
	}
	store, err := tool.OpenReleaseCache()
	if err != nil {
		return "", err
	}
	path, err := store.Ensure(ctx, manager.Spec, manager.Binary)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrManager, manager.Command, err)
	}
	return path, nil
}

// invocation is one package-manager command in a project directory.
type invocation struct {
	directory string
	binary    string
	sentinel  error
}

func (call invocation) run(ctx context.Context, args ...string) error {
	if err := ctx.Err(); err != nil {
		return context.Cause(ctx)
	}
	cmd := execdriver.MustCommand(ctx, call.binary, args...)
	cmd.Dir = call.directory
	cmd.Env = childEnv(call.binary, call.directory)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if err := execdriver.Run(ctx, cmd); err != nil {
		return commandFailed(call.sentinel, log.Bytes(), err)
	}
	return nil
}

func childEnv(binary, directory string) []string {
	env := os.Environ()
	path := os.Getenv("PATH")
	local := filepath.Join(directory, "node_modules", ".bin")
	path = filepath.Dir(binary) + string(os.PathListSeparator) + path + string(os.PathListSeparator) + local
	replaced := false
	for i, item := range env {
		if strings.HasPrefix(item, "PATH=") {
			env[i] = "PATH=" + path
			replaced = true
		}
	}
	if !replaced {
		env = append(env, "PATH="+path)
	}
	return env
}

func commandFailed(sentinel error, log []byte, err error) error {
	text := strings.TrimSpace(string(log))
	if len(text) > maxLog {
		text = text[len(text)-maxLog:]
	}
	if text == "" {
		return fmt.Errorf("%w: %w", sentinel, err)
	}
	return fmt.Errorf("%w: %s: %w", sentinel, text, err)
}
