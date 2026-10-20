package host

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

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

// Job is one or more CGO-free archives.
// AppID is stamped into x/release.appID. Version is stamped into x/release.version.
type Job struct {
	Dir     string
	Out     string
	Name    string
	AppID   string
	Version string
	Targets []build.Target
	// Sign, when set, Authenticode-signs a Windows exe, Mach-O-signs a
	// Darwin binary, and writes a detached CMS signature beside every archive.
	Sign *sign.Identity
}

// Desktop cross-compiles one archive for every release target.
func Desktop(ctx context.Context, job Job) ([]string, error) {
	job.Targets = build.DesktopTargets()
	return job.Run(ctx)
}

func (job Job) project() string {
	if job.Name != "" {
		return job.Name
	}
	return filepath.Base(job.Dir)
}

// Paths is the archive list Run writes, including a .cms file beside
// each archive when Sign is set. The names are known before the build.
func (job Job) Paths() []string {
	project := job.project()
	out := make([]string, 0, len(job.Targets)*2)
	for _, target := range job.Targets {
		archive := filepath.Join(job.Out, build.ArchiveName(project, target))
		out = append(out, archive)
		if job.Sign != nil {
			out = append(out, archive+".cms")
		}
	}
	return out
}

// Run writes one archive for each target.
// Each target is a workflow step named goos/goarch, so the steps run
// together. Run waits on the future for those steps.
func (job Job) Run(ctx context.Context) ([]string, error) {
	if len(job.Targets) == 0 {
		return nil, ErrBuildNoTarget
	}
	if err := os.MkdirAll(job.Out, 0o755); err != nil {
		return nil, err
	}
	fut, err := workflow.Run(ctx, job.graph(), nil)
	if err != nil {
		return nil, err
	}
	if err := fut.Wait(ctx); err != nil {
		return nil, err
	}
	return job.Paths(), nil
}

func (job Job) graph() workflow.Graph {
	project := job.project()
	stamp := version.Info{Version: job.Version, BuiltBy: release.Name()}
	steps := make([]workflow.Step, 0, len(job.Targets))
	defaults := make([]string, 0, len(job.Targets))
	for _, target := range job.Targets {
		target := target
		name := target.GOOS + "/" + target.GOARCH
		defaults = append(defaults, name)
		steps = append(steps, workflow.Step{
			Name:         name,
			Desc:         "build " + name,
			Orchestrator: true,
			Tasks: []workflow.Task{
				workflow.Func(func(ctx context.Context, _ *taskgroup.Status) error {
					return job.archiveTarget(ctx, project, stamp, target)
				}),
			},
		})
	}
	return workflow.Graph{Dir: job.Out, Steps: steps, Defaults: defaults}
}

func (job Job) archiveTarget(ctx context.Context, project string, stamp version.Info, target build.Target) error {
	binary := project
	if target.GOOS == "windows" {
		binary += ".exe"
	}
	tmp, err := os.MkdirTemp("", release.Name()+"-build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	binPath := filepath.Join(tmp, binary)
	slog.Info("build " + target.GOOS + "/" + target.GOARCH)
	err = gocmd.Command{
		Dir:  job.Dir,
		Env:  append(os.Environ(), "CGO_ENABLED=0", "GOOS="+target.GOOS, "GOARCH="+target.GOARCH),
		Args: []string{"-trimpath", "-ldflags", stamp.WithAppID(job.AppID), "-o", binPath, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("%w %s/%s: %w", ErrJobArchiveTargetBuild, target.GOOS, target.GOARCH, err)
	}
	if job.Sign != nil && target.GOOS == "windows" {
		if err := job.Sign.SignPEFile(ctx, binPath, project); err != nil {
			return fmt.Errorf("%w %s/%s: %w", ErrJobArchiveTargetSign2, target.GOOS, target.GOARCH, err)
		}
	}
	if job.Sign != nil && target.GOOS == "darwin" {
		if err := sign.SignMachO(job.Sign, binPath, job.AppID); err != nil {
			return fmt.Errorf("%w %s/%s: %w", ErrJobArchiveTargetSign2, target.GOOS, target.GOARCH, err)
		}
	}
	archive := filepath.Join(job.Out, build.ArchiveName(project, target))
	if err := writeArchive(archive, binPath, binary); err != nil {
		return err
	}
	if job.Sign == nil {
		return nil
	}
	raw, err := os.ReadFile(archive)
	if err != nil {
		return err
	}
	sig, err := job.Sign.SignCMS(raw)
	if err != nil {
		return fmt.Errorf("%w %s: %w", ErrJobArchiveTargetSign, archive, err)
	}
	return os.WriteFile(archive+".cms", sig, 0o644)
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
