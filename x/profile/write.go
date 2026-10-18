package profile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
)

func (p *Profile) file(name string) string {
	return filepath.Join(p.directory, fmt.Sprintf("%s.prof", name))
}

func (p *Profile) write(ctx context.Context) error {
	if err := os.MkdirAll(p.directory, 0o755); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	cpuFile, err := os.Create(p.file("cpu"))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		cpuFile.Close()
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	<-ctx.Done()
	// Stop before the other snapshots. Writing them while the profiler
	// thread is still running can stall process exit on Windows.
	pprof.StopCPUProfile()
	if err := cpuFile.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	for _, prof := range pprof.Profiles() {
		if err := writeProfile(p.file(prof.Name()), prof); err != nil {
			return err
		}
	}
	return nil
}

func writeProfile(path string, prof *pprof.Profile) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	writeErr := prof.WriteTo(file, 0)
	closeErr := file.Close()
	if writeErr != nil {
		return fmt.Errorf("%w: %s: %w", ErrProfileWrite, path, writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("%w: %s: %w", ErrProfileWrite, path, closeErr)
	}
	return nil
}
