package tool

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	stdpath "path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/report"
	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/fs/squashfs"
	tarfs "github.com/lewtec/lewkit/x/fs/tar"
	zipfs "github.com/lewtec/lewkit/x/fs/zip"
	lewpath "github.com/lewtec/lewkit/x/path"
)

var (
	// ErrEmptyDownloadURL is returned when a download URL is blank.
	ErrEmptyDownloadURL = errors.New("download URL cannot be empty")
	// ErrNoDownloadURLs is returned when every candidate URL is blank.
	ErrNoDownloadURLs = errors.New("no download URLs provided")
)

// DownloadOptions controls one file download.
// Mode 0 leaves the process umask in place. Hash empty skips verification.
type DownloadOptions struct {
	Hash             string
	Size             int64
	Mode             os.FileMode
	ConfigureRequest func(*http.Request)
}

// InstallArtifact downloads artifact, extracts it into destination, strips a
// single top-level directory, and renames platform-qualified binaries.
func InstallArtifact(ctx context.Context, artifact Artifact, destination string, options DownloadOptions) error {
	if options.Hash == "" {
		options.Hash = artifact.Hash
	}
	if options.Size <= 0 {
		options.Size = artifact.Size
	}

	temporary := destination + ".download"
	if err := os.MkdirAll(temporary, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(temporary)

	downloadPath := filepath.Join(temporary, stdpath.Base(artifact.URL))
	if err := DownloadFile(ctx, artifact.URL, downloadPath, options); err != nil {
		return err
	}

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	if err := Extract(ctx, downloadPath, destination); err != nil {
		return fmt.Errorf("extract %s: %w", stdpath.Base(artifact.URL), err)
	}
	return NormalizeInstalledBinaries(destination)
}

// DownloadFile writes url to destination and checks options.Hash when set.
func DownloadFile(ctx context.Context, url, destination string, options DownloadOptions) error {
	if strings.TrimSpace(url) == "" {
		return ErrEmptyDownloadURL
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := downloadDirect(ctx, url, destination, options); err != nil {
		return err
	}
	if err := verifyHash(destination, options.Hash); err != nil {
		if removeErr := os.Remove(destination); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove mismatched download", "error", removeErr, "path", destination)
		}
		return err
	}
	return nil
}

// DownloadFirst tries urls in order and returns the first success.
func DownloadFirst(ctx context.Context, urls []string, destination string, options DownloadOptions) error {
	var failures []string
	for _, url := range urls {
		if strings.TrimSpace(url) == "" {
			continue
		}
		if err := DownloadFile(ctx, url, destination, options); err == nil {
			return nil
		} else {
			failures = append(failures, fmt.Sprintf("%s: %v", url, err))
		}
	}
	if len(failures) == 0 {
		return ErrNoDownloadURLs
	}
	return fmt.Errorf("all downloads failed: %s", strings.Join(failures, "; "))
}

func downloadDirect(ctx context.Context, url, destination string, options DownloadOptions) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if options.ConfigureRequest != nil {
		options.ConfigureRequest(request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { report.Report(response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("GET %s: %s", url, response.Status)
		if response.StatusCode == http.StatusForbidden {
			err = fmt.Errorf("%w (if this is a GitHub release asset, set GITHUB_TOKEN or run 'gh auth login' to increase rate limits)", err)
		}
		return err
	}

	temporary, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	success := false
	defer func() {
		if !success {
			os.Remove(temporaryPath)
		}
	}()
	if _, err := io.Copy(temporary, response.Body); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if options.Mode != 0 {
		if err := os.Chmod(temporaryPath, options.Mode); err != nil {
			return err
		}
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	success = true
	return nil
}

func verifyHash(path, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	algorithm, sum := "sha256", raw
	if left, right, ok := strings.Cut(raw, ":"); ok {
		algorithm, sum = strings.ToLower(left), right
	}
	if algorithm != "sha256" {
		return fmt.Errorf("unsupported hash algorithm %q", algorithm)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return err
	}
	got := hex.EncodeToString(digest.Sum(nil))
	if !strings.EqualFold(got, sum) {
		return fmt.Errorf("hash mismatch for %s", path)
	}
	return nil
}

// Extract unpacks a zip, squashfs, or tar archive into destination.
// A single top directory is removed by [lewfs.StripTopDirectory], then [lewfs.Copy] writes that filesystem.
// A file that is none of those archives is copied in as one binary.
func Extract(ctx context.Context, source, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	dest, err := lewpath.Open(destination)
	if err != nil {
		return err
	}
	defer dest.Close()

	file, err := openHostFile(source)
	if err != nil {
		return err
	}
	defer file.Close()

	var archive fs.FS
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if opened, err := zipfs.Open(ctx, file); err == nil {
		archive = opened
	} else if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	} else if opened, err := squashfs.Open(ctx, file); err == nil {
		archive = opened
	} else if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	} else if opened, err := tarfs.Open(ctx, file); err == nil {
		archive = opened
	} else if errors.Is(err, fs.ErrInvalid) {
		return err
	} else {
		return installBinary(ctx, source, dest)
	}
	stripped, err := lewfs.StripTopDirectory(archive)
	if err != nil {
		return err
	}
	return lewfs.Copy(ctx, dest, lewfs.Walk(ctx, stripped, nil))
}

type hostFile struct {
	*os.File
	root *lewpath.Root
}

func (file *hostFile) Close() error {
	return errors.Join(file.File.Close(), file.root.Close())
}

func openHostFile(osPath string) (*hostFile, error) {
	parent, err := lewpath.Open(filepath.Dir(osPath))
	if err != nil {
		return nil, err
	}
	opened, err := lewpath.New(filepath.Base(osPath)).Open(parent)
	if err != nil {
		parent.Close()
		return nil, err
	}
	file, ok := opened.(*os.File)
	if !ok {
		opened.Close()
		parent.Close()
		return nil, fmt.Errorf("open %s: not an os file", osPath)
	}
	return &hostFile{File: file, root: parent}, nil
}

func installBinary(ctx context.Context, source string, dest *lewpath.Root) error {
	input, err := openHostFile(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	name := lewpath.New(NormalizeBinaryName(stdpath.Base(source)))
	listing := func(yield func(lewfs.File, error) bool) {
		yield(lewfs.File{
			Name:   name,
			Mode:   0o755,
			Size:   info.Size(),
			Reader: input,
		}, nil)
	}
	return lewfs.Copy(ctx, dest, listing)
}

var binaryNameSuffixes = []string{
	"-x86_64-unknown-linux-musl", "-aarch64-unknown-linux-musl",
	"-x86_64-unknown-linux-gnu", "-aarch64-unknown-linux-gnu",
	"-x86_64-apple-darwin", "-aarch64-apple-darwin",
	"-x86_64-pc-windows-msvc", "-aarch64-pc-windows-msvc",
	"-linux-amd64", "-linux-x86_64", "-linux-x64",
	"-linux-arm64", "-linux-aarch64",
	"-linux-386", "-linux-x86",
	"_linux_amd64", "_linux_arm64", "_linux_x86_64", "_linux_x64",
	"_linux_386", "_linux_arm", "_linux_mips", "_linux_mips64", "_linux_mips64le",
	"_linux_mipsle", "_linux_s390x",
	".linux.amd64", ".linux.arm64", ".linux.x86_64", ".linux.x64",
	"-darwin-amd64", "-darwin-x86_64", "-darwin-x64",
	"-darwin-arm64", "-darwin-aarch64",
	"_darwin_amd64", "_darwin_arm64", "_darwin_386",
	".darwin.amd64", ".darwin.arm64", ".darwin.x86_64",
	"-windows-amd64", "-windows-x86_64", "-windows-x64",
	"-windows-arm64",
	"-windows-386", "-windows-x86",
	"_windows_amd64", "_windows_386",
	".windows.amd64", ".windows.arm64",
	"_freebsd_amd64", "_freebsd_386", "_freebsd_arm",
	"_netbsd_amd64", "_netbsd_386", "_netbsd_arm",
	"_openbsd_amd64", "_openbsd_386",
	"-linux", "-darwin", "-macos", "-windows",
	"_linux", "_darwin", "_macos", "_windows",
	".linux", ".darwin", ".macos", ".windows",
	"-amd64", "-x86_64", "-x64",
	"-arm64", "-aarch64",
	"-386", "-x86",
	"_amd64", "_arm64", "_x86_64", "_x64", "_386",
	".amd64", ".arm64", ".x86_64", ".x64",
}

var binaryVersionPattern = regexp.MustCompile(`[-_](v?\d+\.[\d.]+\w*)$`)

// NormalizeBinaryName strips one platform suffix and a trailing version token.
func NormalizeBinaryName(name string) string {
	result := name
	for _, suffix := range binaryNameSuffixes {
		if before, ok := strings.CutSuffix(result, suffix); ok {
			result = before
			break
		}
	}
	return binaryVersionPattern.ReplaceAllString(result, "")
}

// NormalizeInstalledBinaries renames executables in destination and destination/bin
// whose names still carry a platform triple or version suffix.
func NormalizeInstalledBinaries(destination string) error {
	root, err := lewpath.Open(destination)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, directory := range []lewpath.Path{lewpath.New("."), lewpath.New("bin")} {
		for name, err := range directory.IterDir(root) {
			if err != nil {
				if errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist) {
					break
				}
				return err
			}
			isDirectory, err := name.IsDir(root)
			if err != nil || isDirectory {
				continue
			}
			info, err := name.Stat(root)
			if err != nil {
				return err
			}
			if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
				continue
			}
			newName := NormalizeBinaryName(name.Name())
			if newName == "" || newName == name.Name() {
				continue
			}
			renamed := directory.Join(newName)
			exists, err := renamed.Exists(root)
			if err != nil || exists {
				continue
			}
			if err := name.Rename(root, renamed); err != nil {
				return err
			}
		}
	}
	return nil
}
