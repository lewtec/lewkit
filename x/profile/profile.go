package profile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime/pprof"
)

var (
	ErrProfileWrite = errors.New("can't write profile")
)

func NewProfile(outputDirectory string) Profile {
	return Profile{outputDirectory: outputDirectory}
}

type Profile struct {
	outputDirectory string
}

func (p *Profile) file(name string) string {
	return path.Join(p.outputDirectory, name)
}

func (p *Profile) Run(ctx context.Context) error {
	if err := os.Mkdir(p.outputDirectory, 0o755); err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	fcpu, err := os.Create(p.file("cpu.prof"))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrProfileWrite, err)
	}
	pprof.StartCPUProfile(fcpu)
	defer pprof.StopCPUProfile()
	defer fcpu.Close()

	for _, prof := range pprof.Profiles() {
		fprof, err := os.Create(p.file(fmt.Sprintf("%s.prof", prof.Name())))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrProfileWrite, err)
		}
		defer prof.WriteTo(fprof, 0)
		defer fprof.Close()
	}
	<-ctx.Done()
	return nil
}
