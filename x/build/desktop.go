package build

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/version"
)

// Target is one CGO-free desktop binary.
type Target struct {
	GOOS   string
	GOARCH string
}

// DesktopTargets is the release matrix: Linux, macOS, and Windows on amd64 and arm64.
func DesktopTargets() []Target {
	var out []Target
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			out = append(out, Target{GOOS: goos, GOARCH: goarch})
		}
	}
	return out
}

// ArchiveName is Name_OS_arch.tar.gz, or .zip on Windows.
func ArchiveName(project string, target Target) string {
	arch := target.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	ext := ".tar.gz"
	if target.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s_%s_%s%s", project, title(target.GOOS), arch, ext)
}

func title(goos string) string {
	if goos == "" {
		return ""
	}
	return strings.ToUpper(goos[:1]) + goos[1:]
}

// Job is one or more CGO-free archives. Context is the parent taskgroup context.
// AppID is stamped into x/release.appID. Version is stamped into x/release.version.
type Job struct {
	Context context.Context
	Dir     string
	Out     string
	Name    string
	AppID   string
	Version string
	Targets []Target
}

// Desktop cross-compiles one archive for every release target.
func Desktop(job Job) ([]string, error) {
	job.Targets = DesktopTargets()
	return Archives(job)
}

// Archives writes one archive for each target.
func Archives(job Job) ([]string, error) {
	ctx := job.Context
	if ctx == nil {
		ctx = context.Background()
	}
	project := job.Name
	if project == "" {
		project = filepath.Base(job.Dir)
	}
	if err := os.MkdirAll(job.Out, 0o755); err != nil {
		return nil, err
	}
	stamp := version.Info{Version: job.Version, BuiltBy: "lewkit"}
	var written []string
	for _, target := range job.Targets {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		binary := project
		if target.GOOS == "windows" {
			binary += ".exe"
		}
		tmp, err := os.MkdirTemp("", "lewkit-build-")
		if err != nil {
			return written, err
		}
		binPath := filepath.Join(tmp, binary)
		slog.Info("build " + target.GOOS + "/" + target.GOARCH)
		err = gocmd.Command{
			Context: ctx,
			Dir:     job.Dir,
			Env:     append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.GOOS, "GOARCH="+target.GOARCH),
			Args:    []string{"-trimpath", "-ldflags", stamp.WithAppID(job.AppID), "-o", binPath, "."},
		}.Run()
		if err != nil {
			os.RemoveAll(tmp)
			return written, fmt.Errorf("build %s/%s: %w", target.GOOS, target.GOARCH, err)
		}
		archive := filepath.Join(job.Out, ArchiveName(project, target))
		if err := writeArchive(archive, binPath, binary); err != nil {
			os.RemoveAll(tmp)
			return written, err
		}
		os.RemoveAll(tmp)
		written = append(written, archive)
	}
	return written, nil
}

func writeArchive(archive, binPath, name string) error {
	if strings.HasSuffix(archive, ".zip") {
		return writeZip(archive, binPath, name)
	}
	return writeTarGz(archive, binPath, name)
}

func writeTarGz(archive, binPath, name string) error {
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	return addFile(tw, binPath, name)
}

func addFile(tw *tar.Writer, path, name string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(tw, file)
	return err
}

func writeZip(archive, binPath, name string) error {
	out, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	info, err := os.Stat(binPath)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate
	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(binPath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}
