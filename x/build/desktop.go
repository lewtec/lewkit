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

// Job is one or more CGO-free archives.
// AppID is stamped into x/release.appID. Version is stamped into x/release.version.
type Job struct {
	Dir     string
	Out     string
	Name    string
	AppID   string
	Version string
	Targets []Target
}

// Desktop cross-compiles one archive for every release target.
func Desktop(ctx context.Context, job Job) ([]string, error) {
	job.Targets = DesktopTargets()
	return job.Run(ctx)
}

// Run writes one archive for each target.
func (job Job) Run(ctx context.Context) ([]string, error) {
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
		err = goBinary{
			dir:     job.Dir,
			appID:   job.AppID,
			version: job.Version,
			goos:    target.GOOS,
			goarch:  target.GOARCH,
			dest:    binPath,
		}.compile(ctx)
		if err != nil {
			os.RemoveAll(tmp)
			return written, err
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

// goBinary is one CGO-free go build. windowsGUI sets the PE subsystem to
// IMAGE_SUBSYSTEM_WINDOWS_GUI so opening the exe does not create a console.
type goBinary struct {
	dir, appID, version, goos, goarch, dest string
	windowsGUI                              bool
}

func (b goBinary) compile(ctx context.Context) error {
	// -o is relative to Cmd.Dir, which is the module, not the caller's cwd.
	dest, err := filepath.Abs(b.dest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	stamp := version.Info{Version: b.version, BuiltBy: "lewkit"}.WithAppID(b.appID)
	if b.windowsGUI {
		stamp += " -H windowsgui"
	}
	slog.Info("build " + b.goos + "/" + b.goarch)
	err = gocmd.Command{
		Dir:  b.dir,
		Env:  append(os.Environ(), "CGO_ENABLED=0", "GOOS="+b.goos, "GOARCH="+b.goarch),
		Args: []string{"-trimpath", "-ldflags", stamp, "-o", dest, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("build %s/%s: %w", b.goos, b.goarch, err)
	}
	return nil
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
