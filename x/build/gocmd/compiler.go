package gocmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/taskgroup"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/lewtec/lewkit/x/toolchain/load"
)

// Compiler selects the go command or the embedded compiler.
// The zero value runs the host go command.
//
//lewkit:enum
type Compiler int

const (
	// CompilerHost runs the go command.
	CompilerHost Compiler = iota + 1
	// CompilerLewkit compiles a CGO-free amd64 or arm64 binary with the embedded compiler.
	CompilerLewkit
)

type compilerKey struct{}

// WithCompiler returns ctx that carries the compiler release builds use.
func WithCompiler(ctx context.Context, c Compiler) context.Context {
	return context.WithValue(ctx, compilerKey{}, c)
}

func (c Command) active(ctx context.Context) Compiler {
	if c.Compiler != 0 {
		return c.Compiler
	}
	got, _ := ctx.Value(compilerKey{}).(Compiler)
	return got
}

func (c Command) runLewkit(ctx context.Context) error {
	verb := c.Verb
	if verb == "" {
		verb = "build"
	}
	if verb != "build" {
		return fmt.Errorf("%w: %s", ErrCompiler, verb)
	}
	parsed, err := parseBuildArgs(c.Args)
	if err != nil {
		return err
	}
	if parsed.buildmode != "" && parsed.buildmode != "exe" {
		return fmt.Errorf("%w: -buildmode=%s", ErrCompiler, parsed.buildmode)
	}
	env := c.Env
	if env == nil {
		env = os.Environ()
	}
	if lookupEnv(env, "CGO_ENABLED") == "1" {
		return fmt.Errorf("%w: CGO_ENABLED=1", ErrCompiler)
	}
	goos := lookupEnv(env, "GOOS")
	goarch := lookupEnv(env, "GOARCH")
	if goarch != "" && goarch != "amd64" && goarch != "arm64" {
		return fmt.Errorf("%w: GOARCH=%s", ErrCompiler, goarch)
	}
	ldflags, err := load.SplitFlags(parsed.ldflags)
	if err != nil {
		return err
	}
	if taskgroup.FromContext(ctx) == nil {
		return c.buildLewkit(ctx, parsed, goos, goarch, ldflags)
	}
	errCh := make(chan error, 1)
	taskgroup.Go(ctx, "go build", taskgroup.Control, func(ctx context.Context, st *taskgroup.Status) error {
		st.Update("lewkit")
		err := c.buildLewkit(ctx, parsed, goos, goarch, ldflags)
		errCh <- err
		return err
	})
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c Command) buildLewkit(ctx context.Context, parsed parsedBuild, goos, goarch string, ldflags []string) error {
	if parsed.output == "" {
		return fmt.Errorf("%w: -o is required", ErrCompiler)
	}
	return gotool.BuildModule(ctx, gotool.Module{
		Dir:      c.Dir,
		Out:      parsed.output,
		GOOS:     goos,
		GOARCH:   goarch,
		Patterns: parsed.pkgs,
		Tags:     load.SplitTags(parsed.tags),
		Ldflags:  ldflags,
		Pkgdir:   parsed.pkgdir,
		Trimpath: parsed.trimpath,
		Toolexec: parsed.toolexec,
	})
}

type parsedBuild struct {
	output    string
	ldflags   string
	tags      string
	pkgdir    string
	pkgs      []string
	trimpath  bool
	buildmode string
	toolexec  string
}

func parseBuildArgs(args []string) (parsedBuild, error) {
	var p parsedBuild
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "" {
			continue
		}
		if a == "--" || !strings.HasPrefix(a, "-") {
			if a == "--" {
				return p, fmt.Errorf("%w: %s", ErrCompiler, a)
			}
			p.pkgs = append(p.pkgs, a)
			continue
		}
		if strings.HasPrefix(a, "--") {
			return p, fmt.Errorf("%w: %s", ErrCompiler, a)
		}
		key, val, has := strings.Cut(strings.TrimPrefix(a, "-"), "=")
		take := func() (string, error) {
			if has {
				return val, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%w: %s needs a value", ErrCompiler, key)
			}
			return args[i], nil
		}
		switch key {
		case "trimpath", "v":
			if has {
				return p, fmt.Errorf("%w: %s", ErrCompiler, a)
			}
			if key == "trimpath" {
				p.trimpath = true
			}
		case "o":
			v, err := take()
			if err != nil {
				return p, err
			}
			p.output = v
		case "ldflags":
			v, err := take()
			if err != nil {
				return p, err
			}
			p.ldflags = v
		case "tags":
			v, err := take()
			if err != nil {
				return p, err
			}
			p.tags = v
		case "pkgdir":
			v, err := take()
			if err != nil {
				return p, err
			}
			p.pkgdir = v
		case "buildmode":
			v, err := take()
			if err != nil {
				return p, err
			}
			p.buildmode = v
		case "toolexec":
			v, err := take()
			if err != nil {
				return p, err
			}
			export, ok := cgoExportArg(v)
			if !ok {
				return p, fmt.Errorf("%w: -toolexec", ErrCompiler)
			}
			if _, err := os.Stat(export); err != nil {
				return p, fmt.Errorf("%w: -toolexec: %s", ErrCompiler, err)
			}
			p.toolexec = v
		default:
			return p, fmt.Errorf("%w: %s", ErrCompiler, a)
		}
	}
	return p, nil
}

// cgoExportArg reports the _cgo_*_export.go path inside one -toolexec
// value. cmd/go splits that value the way a shell would. The basename
// must start with _cgo_ and end with _export.go. That covers
// _cgo_android_export.go and _cgo_clib_export.go. Any other wrapper
// stays rejected: this compiler does not run it.
func cgoExportArg(val string) (string, bool) {
	words, err := splitToolexecWords(val)
	if err != nil {
		return "", false
	}
	for _, w := range words {
		base := filepath.Base(w)
		if strings.HasPrefix(base, "_cgo_") && strings.HasSuffix(base, "_export.go") {
			return w, true
		}
	}
	return "", false
}

func splitToolexecWords(s string) ([]string, error) {
	var words []string
	var b strings.Builder
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if b.Len() > 0 {
				words = append(words, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("%w: -toolexec", ErrCompiler)
	}
	if b.Len() > 0 {
		words = append(words, b.String())
	}
	return words, nil
}

func lookupEnv(env []string, key string) string {
	prefix := key + "="
	val := ""
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			val = e[len(prefix):]
		}
	}
	return val
}
