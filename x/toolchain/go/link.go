package gotool

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/sys"
	"github.com/lewtec/lewkit/x/toolchain/go/src/link/amd64"
	"github.com/lewtec/lewkit/x/toolchain/go/src/link/arm64"
	"github.com/lewtec/lewkit/x/toolchain/go/src/link/ld"
	"github.com/lewtec/lewkit/x/toolchain/go/src/std/buildcfg"
)

// Link links archive as one image. mode is exe, pie, or c-shared.
// ldflags are extra linker arguments, such as the tamago entry and text
// address. The link runs in its own process.
func Link(ctx context.Context, out, importcfg, mode, archive string, ldflags ...string) error {
	return spawn(ctx, "link", childRequest{
		Out:       out,
		ImportCfg: importcfg,
		Mode:      mode,
		Archive:   archive,
		Ldflags:   ldflags,
	})
}

// linkHere is the link inside the process spawn started.
// One process links one image, then exits.
func linkHere(out, importcfg, mode, archive string, ldflags ...string) (err error) {
	arch, theArch, errArch := linkArch(buildcfg.GOARCH)
	if errArch != nil {
		return errArch
	}
	args := []string{
		"-o", out,
		"-importcfg", importcfg,
		"-buildmode=" + mode,
		"-linkmode=internal",
	}
	args = append(args, ldflags...)
	args = append(args, archive)

	var buf bytes.Buffer
	ld.SetDiag(&buf)
	defer ld.SetDiag(nil)
	defer func() {
		msg := buf.String()
		rec := recover()
		if rec == nil {
			if _, statErr := os.Stat(out); statErr != nil {
				err = fmt.Errorf("link wrote no file: %w\n%s", statErr, msg)
			}
			return
		}
		sig, ok := rec.(ld.ExitSignal)
		if !ok {
			panic(rec)
		}
		if sig != 0 {
			err = fmt.Errorf("link exited %d\n%s", int(sig), msg)
			return
		}
		if _, statErr := os.Stat(out); statErr != nil {
			err = fmt.Errorf("link wrote no file: %w\n%s", statErr, msg)
		}
	}()

	ld.Run(arch, theArch, args)
	return nil
}

func linkArch(goarch string) (*sys.Arch, ld.Arch, error) {
	switch goarch {
	case "amd64":
		arch, theArch := amd64.Init()
		return arch, theArch, nil
	case "arm64":
		arch, theArch := arm64.Init()
		return arch, theArch, nil
	default:
		return nil, ld.Arch{}, fmt.Errorf("%w %s", ErrLinkSupportsGoarchAmd64OrArm64Have, goarch)
	}
}
