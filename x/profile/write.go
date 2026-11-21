package profile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"

	"github.com/lewtec/lewkit/x/io"
)

func (p *Profile) file(name string) string {
	return filepath.Join(p.directory, fmt.Sprintf("%s.prof", name))
}

func (p *Profile) write(ctx context.Context) error {
	if err := io.Mkdirp(p.directory); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	cpuFile, err := os.Create(p.file("cpu"))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	defer cpuFile.Close()
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	defer pprof.StopCPUProfile()

	for _, prof := range pprof.Profiles() {
		profileFile, err := os.Create(p.file(prof.Name()))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrProfileWrite, err)
		}
		defer profileFile.Close()
		defer prof.WriteTo(profileFile, 0)
	}
	<-ctx.Done()
	return nil
}
