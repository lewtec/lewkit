package tool

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/io/atomic"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/taskgroup"
)

var (
	// ErrNoVersionsFound is returned when a backend lists no versions.
	ErrNoVersionsFound = errors.New("no versions found")
	// ErrToolDirectoryNotFound is returned when a version directory is missing.
	ErrToolDirectoryNotFound = errors.New("tool directory not found")
	// ErrEmptyStore is returned when Open is given a blank root.
	ErrEmptyStore = errors.New("tool store root is empty")
)

// Store maps specs onto version directories under directory.
type Store struct {
	directory string
}

// Installed is one version directory present on disk.
type Installed struct {
	Name    string
	Version string
	Path    string
}

// Open binds root as the store directory. root is not created until Install.
func Open(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, ErrEmptyStore
	}
	return &Store{directory: root}, nil
}

// Install fetches specification into the store when that version directory is missing or empty.
func (store *Store) Install(ctx context.Context, specification string) error {
	return store.install(ctx, specification, "")
}

// Ensure installs specification when needed and returns the absolute path of binaryName.
func (store *Store) Ensure(ctx context.Context, specification, binaryName string) (string, error) {
	spec, err := Parse(specification)
	if err != nil {
		return "", err
	}
	actualVersion := spec.Version
	if spec.Version == "latest" {
		resolved, err := store.resolveLatest(ctx, spec)
		if err != nil {
			return "", fmt.Errorf("resolve latest version: %w", err)
		}
		actualVersion = resolved
		spec.Version = actualVersion
	}
	backend, err := Get(spec.Backend)
	if err != nil {
		return "", err
	}
	installed, err := backend.Tool(spec.Package)
	if err != nil {
		return "", err
	}

	normalized := normalizeVersion(actualVersion)
	versionName := versionPath(spec, actualVersion)
	versionDirectory := store.host(versionName)

	noCache := NoCache(ctx)
	hasFiles, err := store.versionHasFiles(versionName)
	if err != nil {
		return "", err
	}
	if hasFiles && !noCache {
		if err := fixAndCheck(ctx, installed, versionDirectory); err != nil {
			slog.InfoContext(ctx, "existing install failed checks; reinstalling", "spec", spec.String(), "error", err)
		} else if binaryPath := FindBinary(versionDirectory, binaryName); binaryPath != "" {
			return binaryPath, nil
		} else {
			return "", fmt.Errorf("%w: %q in %s", ErrBinaryNotFound, binaryName, versionDirectory)
		}
	}
	if noCache && DryRun(ctx) {
		if binaryPath := FindBinary(versionDirectory, binaryName); binaryPath != "" {
			slog.DebugContext(ctx, "no-cache: would reinstall tool", "spec", spec.String(), "binary", binaryName)
			return binaryPath, nil
		}
	}
	if noCache {
		slog.DebugContext(ctx, "no-cache: reinstalling tool", "spec", spec.String(), "binary", binaryName)
	}

	slog.InfoContext(ctx, "installing tool", "spec", spec.String(), "backend", spec.Backend, "version", actualVersion, "binary", binaryName)
	if binaryTool, ok := installed.(BinaryTool); ok {
		return store.ensureBinaryTool(ctx, spec, installed, binaryTool, actualVersion, normalized, versionDirectory, binaryName)
	}
	if err := store.install(ctx, spec.String(), binaryName); err != nil {
		return "", fmt.Errorf("install tool: %w", err)
	}
	binaryPath, err := store.lookup(spec, binaryName)
	if err != nil {
		return "", fmt.Errorf("tool installed but binary not found: %w", err)
	}
	slog.InfoContext(ctx, "tool installed", "spec", spec.String(), "path", binaryPath)
	return binaryPath, nil
}

func (store *Store) ensureBinaryTool(ctx context.Context, spec Spec, installed Tool, binaryTool BinaryTool, actualVersion, normalized, versionDirectory, binaryName string) (string, error) {
	operation := atomic.NewOperation(versionDirectory, true)
	workPath := operation.StagingPath()
	defer func() {
		if removeErr := operation.Rollback(); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
		}
	}()
	if err := os.MkdirAll(workPath, 0o755); err != nil {
		return "", err
	}
	var binaryPath string
	installErr := taskgroup.GoIsolated(ctx, "install:"+spec.String(), taskgroup.Control, func(ctx context.Context, status *taskgroup.Status) error {
		status.Update("installing " + normalized)
		var err error
		binaryPath, err = binaryTool.EnsureBinary(ctx, actualVersion, binaryName, workPath)
		return err
	})
	if installErr != nil {
		return "", fmt.Errorf("install tool: %w", installErr)
	}
	if err := fixAndCheck(ctx, installed, workPath); err != nil {
		return "", err
	}
	if err := operation.Commit(); err != nil {
		return "", fmt.Errorf("install swap %s: %w", versionDirectory, err)
	}
	binaryPath = FindBinary(versionDirectory, binaryName)
	if binaryPath == "" {
		return "", fmt.Errorf("%w: %q in %s", ErrBinaryNotFound, binaryName, versionDirectory)
	}
	slog.InfoContext(ctx, "tool installed", "spec", spec.String(), "path", binaryPath)
	return binaryPath, nil
}

func (store *Store) install(ctx context.Context, specification, binaryHint string) error {
	slog.DebugContext(ctx, "installing tool", "input", specification)
	spec, err := Parse(specification)
	if err != nil {
		return err
	}
	slog.DebugContext(ctx, "parsed spec", "spec", spec.String())

	backend, err := Get(spec.Backend)
	if err != nil {
		return err
	}
	installed, err := backend.Tool(spec.Package)
	if err != nil {
		return err
	}

	version := spec.Version
	if version == "latest" {
		resolved, err := store.resolveLatest(ctx, spec)
		if err != nil {
			return fmt.Errorf("resolve latest version: %w", err)
		}
		version = resolved
	}
	normalized := normalizeVersion(version)
	finalPath := store.host(versionPath(spec, version))
	operation := atomic.NewOperation(finalPath, true)
	workPath := operation.StagingPath()
	defer func() {
		if removeErr := operation.Rollback(); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
		}
	}()
	if err := os.MkdirAll(workPath, 0o755); err != nil {
		return err
	}

	doInstall := func(ctx context.Context) error {
		if binaryHint != "" {
			if artifactTool, ok := installed.(ArtifactTool); ok {
				artifacts, err := artifactTool.ListArtifacts(ctx, version)
				if err == nil {
					if chosen := SelectArtifact(artifacts, runtime.GOOS, runtime.GOARCH, binaryHint); chosen != nil {
						slog.Debug("installing with artifact hint", "url", chosen.URL, "hint", binaryHint)
						return artifactTool.InstallArtifact(ctx, *chosen, workPath)
					}
				}
			}
		}
		slog.Debug("installing tool via Tool.Install", "destination", workPath)
		return installed.Install(ctx, version, workPath)
	}

	if err := taskgroup.GoIsolated(ctx, "install:"+spec.String(), taskgroup.Control, func(ctx context.Context, status *taskgroup.Status) error {
		status.Update("installing " + normalized)
		return doInstall(ctx)
	}); err != nil {
		return fmt.Errorf("installation: %w", err)
	}
	if err := fixAndCheck(ctx, installed, workPath); err != nil {
		return err
	}
	if err := operation.Commit(); err != nil {
		return fmt.Errorf("install swap %s: %w", finalPath, err)
	}
	slog.InfoContext(ctx, "tool installed", "spec", spec.String(), "version", normalized, "path", finalPath)
	return nil
}

func (store *Store) resolveLatest(ctx context.Context, spec Spec) (string, error) {
	backend, err := Get(spec.Backend)
	if err != nil {
		return "", err
	}
	installed, err := backend.Tool(spec.Package)
	if err != nil {
		return "", err
	}
	versions, err := installed.ListVersions(ctx)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "", ErrNoVersionsFound
	}
	return versions[0], nil
}

func (store *Store) lookup(spec Spec, binaryName string) (string, error) {
	name := versionPath(spec, spec.Version)
	versionDirectory := store.host(name)
	root, err := lewpath.Open(store.directory)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrToolDirectoryNotFound, versionDirectory)
	}
	defer root.Close()
	isDirectory, err := name.IsDir(root)
	if err != nil {
		return "", err
	}
	if !isDirectory {
		return "", fmt.Errorf("%w: %s", ErrToolDirectoryNotFound, versionDirectory)
	}
	if binaryPath := FindBinary(versionDirectory, binaryName); binaryPath != "" {
		return binaryPath, nil
	}
	return "", fmt.Errorf("%w: %q in %s", ErrBinaryNotFound, binaryName, versionDirectory)
}

// ListInstalled returns every version directory under the store. A missing root is an empty list.
func (store *Store) ListInstalled() ([]Installed, error) {
	root, err := lewpath.Open(store.directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer root.Close()
	var installed []Installed
	for packageName, err := range lewpath.New(".").IterDir(root) {
		if err != nil {
			return nil, err
		}
		isDirectory, err := packageName.IsDir(root)
		if err != nil || !isDirectory {
			continue
		}
		for versionName, err := range packageName.IterDir(root) {
			if err != nil {
				return nil, err
			}
			isDirectory, err := versionName.IsDir(root)
			if err != nil || !isDirectory {
				continue
			}
			installed = append(installed, Installed{
				Name:    packageName.Name(),
				Version: versionName.Name(),
				Path:    joinHost(root.Name(), versionName),
			})
		}
	}
	return installed, nil
}

// Resolve finds binaryName among installed trees.
// Version order is a .tool-versions line walking up from the working directory,
// then LEWKIT_<NAME>_VERSION, then the newest installed tree that contains the binary.
func (store *Store) Resolve(ctx context.Context, binaryName string) (string, error) {
	version, err := store.resolvePinnedVersion(binaryName)
	if err != nil {
		return "", err
	}
	root, err := lewpath.Open(store.directory)
	if err != nil {
		return "", err
	}
	defer root.Close()
	var candidates []string
	for packageName, err := range lewpath.New(".").IterDir(root) {
		if err != nil {
			return "", err
		}
		isDirectory, err := packageName.IsDir(root)
		if err != nil || !isDirectory {
			continue
		}
		if version == "latest" {
			var names []string
			for versionName, err := range packageName.IterDir(root) {
				if err != nil {
					return "", err
				}
				isDirectory, err := versionName.IsDir(root)
				if err != nil || !isDirectory {
					continue
				}
				names = append(names, versionName.Name())
			}
			sort.Slice(names, func(i, j int) bool {
				return CompareVersions(names[i], names[j]) < 0
			})
			for i := len(names) - 1; i >= 0; i-- {
				binaryPath := FindBinary(joinHost(root.Name(), packageName.Join(names[i])), binaryName)
				if binaryPath != "" {
					candidates = append(candidates, binaryPath)
					break
				}
			}
			continue
		}
		binaryPath := FindBinary(joinHost(root.Name(), packageName.Join(version)), binaryName)
		if binaryPath != "" {
			candidates = append(candidates, binaryPath)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("%w: %q (version: %s)", ErrBinaryNotFound, binaryName, version)
	}
	return candidates[0], nil
}

func (store *Store) resolvePinnedVersion(binaryName string) (string, error) {
	version, err := versionFromToolVersions(binaryName)
	if err != nil {
		return "", err
	}
	if version != "" {
		return version, nil
	}
	envKey := "LEWKIT_" + strings.ToUpper(strings.ReplaceAll(binaryName, "-", "_")) + "_VERSION"
	if version = os.Getenv(envKey); version != "" {
		return version, nil
	}
	return "latest", nil
}

func versionFromToolVersions(binaryName string) (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", nil
	}
	for {
		version, err := readToolVersion(directory, binaryName)
		if err != nil {
			return "", err
		}
		if version != "" {
			return version, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", nil
}

func readToolVersion(directory, binaryName string) (string, error) {
	root, err := lewpath.Open(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer root.Close()
	name := lewpath.New(".tool-versions")
	isFile, err := name.IsFile(root)
	if err != nil || !isFile {
		return "", err
	}
	body, err := name.ReadFile(root)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == binaryName {
			return fields[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan %s: %w", directory, err)
	}
	return "", nil
}

func versionPath(spec Spec, version string) lewpath.Path {
	return lewpath.New(spec.Directory(), normalizeVersion(version))
}

func (store *Store) host(name lewpath.Path) string {
	return joinHost(store.directory, name)
}

func (store *Store) versionHasFiles(name lewpath.Path) (bool, error) {
	root, err := lewpath.Open(store.directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer root.Close()
	for _, err := range name.IterDir(root) {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func fixAndCheck(ctx context.Context, installed Tool, destination string) error {
	if fixer, ok := installed.(Fixer); ok {
		if err := fixer.Fix(ctx, destination); err != nil {
			slog.WarnContext(ctx, "post-install fix failed", "error", err, "directory", destination)
		}
	}
	if err := RunChecks(ctx, destination, installed); err != nil {
		if removeErr := os.RemoveAll(destination); removeErr != nil {
			slog.WarnContext(ctx, "remove broken install directory", "error", removeErr, "directory", destination)
		}
		return fmt.Errorf("install checks: %w", err)
	}
	return nil
}
