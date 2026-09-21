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

// Store maps specs onto version directories under root.
type Store struct {
	root string
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
	return &Store{root: root}, nil
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
	versionDirectory := filepath.Join(store.root, spec.Directory(), normalized)

	noCache := NoCache(ctx)
	if entries, err := os.ReadDir(versionDirectory); err == nil && len(entries) > 0 && !noCache {
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
	workPath := versionDirectory + ".tmp"
	if removeErr := os.RemoveAll(workPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
	}
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
		if removeErr := os.RemoveAll(workPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
		}
		return "", fmt.Errorf("install tool: %w", installErr)
	}
	if err := fixAndCheck(ctx, installed, workPath); err != nil {
		if removeErr := os.RemoveAll(workPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
		}
		return "", err
	}
	if err := replaceDirectory(versionDirectory, workPath); err != nil {
		if removeErr := os.RemoveAll(workPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
		}
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
	finalPath := filepath.Join(store.root, spec.Directory(), normalized)
	workPath := finalPath + ".tmp"
	if err := os.RemoveAll(workPath); err != nil {
		return err
	}
	if err := os.MkdirAll(workPath, 0o755); err != nil {
		return err
	}
	cleanupWork := true
	defer func() {
		if cleanupWork {
			if removeErr := os.RemoveAll(workPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				slog.WarnContext(ctx, "remove install work directory", "error", removeErr, "path", workPath)
			}
		}
	}()

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
	if err := replaceDirectory(finalPath, workPath); err != nil {
		return fmt.Errorf("install swap %s: %w", finalPath, err)
	}
	cleanupWork = false
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
	versionDirectory := filepath.Join(store.root, spec.Directory(), normalizeVersion(spec.Version))
	if _, err := os.Stat(versionDirectory); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("%w: %s", ErrToolDirectoryNotFound, versionDirectory)
	}
	if binaryPath := FindBinary(versionDirectory, binaryName); binaryPath != "" {
		return binaryPath, nil
	}
	return "", fmt.Errorf("%w: %q in %s", ErrBinaryNotFound, binaryName, versionDirectory)
}

// ListInstalled returns every version directory under the store. A missing root is an empty list.
func (store *Store) ListInstalled() ([]Installed, error) {
	entries, err := os.ReadDir(store.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var installed []Installed
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		toolPath := filepath.Join(store.root, entry.Name())
		versions, err := os.ReadDir(toolPath)
		if err != nil {
			continue
		}
		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			installed = append(installed, Installed{
				Name:    entry.Name(),
				Version: version.Name(),
				Path:    filepath.Join(toolPath, version.Name()),
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
	entries, err := os.ReadDir(store.root)
	if err != nil {
		return "", err
	}
	var candidates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packageDirectory := filepath.Join(store.root, entry.Name())
		if version == "latest" {
			versions, err := os.ReadDir(packageDirectory)
			if err != nil {
				continue
			}
			var names []string
			for _, versionEntry := range versions {
				if versionEntry.IsDir() {
					names = append(names, versionEntry.Name())
				}
			}
			sort.Slice(names, func(i, j int) bool {
				return CompareVersions(names[i], names[j]) < 0
			})
			for i := len(names) - 1; i >= 0; i-- {
				if binaryPath := FindBinary(filepath.Join(packageDirectory, names[i]), binaryName); binaryPath != "" {
					candidates = append(candidates, binaryPath)
					break
				}
			}
			continue
		}
		if binaryPath := FindBinary(filepath.Join(packageDirectory, version), binaryName); binaryPath != "" {
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
		path := filepath.Join(directory, ".tool-versions")
		if _, err := os.Stat(path); err == nil {
			version, err := readToolVersion(path, binaryName)
			if err != nil {
				return "", err
			}
			if version != "" {
				return version, nil
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", nil
}

func readToolVersion(path, binaryName string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
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
		return "", fmt.Errorf("scan %s: %w", path, err)
	}
	return "", nil
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

func replaceDirectory(destination, temporary string) error {
	backup := destination + ".old"
	if err := os.RemoveAll(backup); err != nil {
		return err
	}
	if _, err := os.Stat(destination); err == nil {
		if err := os.Rename(destination, backup); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(temporary, destination); err != nil {
		if _, statErr := os.Stat(backup); statErr == nil {
			if restoreErr := os.Rename(backup, destination); restoreErr != nil {
				return errors.Join(err, restoreErr)
			}
		}
		return err
	}
	return os.RemoveAll(backup)
}
