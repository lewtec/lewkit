package shader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl/reactor"
	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

var (
	ErrWritePackage = errors.New("")
)

var generatedBy = generate.Notice("shader")

const glslPath = "github.com/lewtec/lewkit/x/ffi/wasm/glsl"

func writePackage(ctx context.Context, dir string, files []shaderFile) error {
	pkg, err := goPackage(dir)
	if err != nil {
		return err
	}
	if !token.IsIdentifier(pkg) {
		return fmt.Errorf("%w: %s", errNoPackage, pkg)
	}
	type compiled struct {
		rel   string
		stage glsl.Stage
		hash  [32]byte
		spirv []byte
	}
	out := make([]compiled, 0, len(files))
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		src, err := os.ReadFile(file.abs)
		if err != nil {
			return err
		}
		spirv, err := reactor.CompileGlslang(ctx, file.stage, src)
		if err != nil {
			return fmt.Errorf("%w%s: %w", ErrWritePackage, file.rel, err)
		}
		out = append(out, compiled{
			rel:   file.rel,
			stage: file.stage,
			hash:  glsl.Hash(file.stage, src),
			spirv: spirv,
		})
	}
	f := golang.NewFile(pkg)
	f.Header = "// " + generatedBy
	for _, file := range out {
		f.Func().Comment(file.rel).Id("init").Block(
			golang.Qual(glslPath, "MustRegisterHash").Call(
				stageCode(file.stage),
				hexComposite(golang.Array(golang.Lit(32), golang.Named("byte")), file.hash[:]),
				hexComposite(golang.Slice(golang.Named("byte")), file.spirv),
			),
		)
	}
	var body bytes.Buffer
	if err := f.Make(&body); err != nil {
		return fmt.Errorf("%w%s: %w", ErrWritePackage, generatedName, err)
	}
	return generate.WriteFile(filepath.Join(dir, generatedName), body.Bytes())
}

func stageCode(stage glsl.Stage) *golang.Expr {
	switch stage {
	case glsl.StageVertex:
		return golang.Qual(glslPath, "StageVertex")
	case glsl.StageFragment:
		return golang.Qual(glslPath, "StageFragment")
	default:
		return golang.Qual(glslPath, "StageCompute")
	}
}

func hexComposite(typ *golang.Type, raw []byte) *golang.Expr {
	elts := make([]*golang.Element, len(raw))
	for i, b := range raw {
		elts[i] = golang.Elem(&golang.Expr{Kind: golang.ExprLit, Lit: golang.LitInt, Text: fmt.Sprintf("0x%02x", b)})
	}
	return golang.Composite(typ, elts...)
}

func stageToken(stage glsl.Stage) string {
	switch stage {
	case glsl.StageVertex:
		return "vertex"
	case glsl.StageFragment:
		return "fragment"
	default:
		return "compute"
	}
}

func ident(rel string) string {
	stem, stage, ok := splitShader(rel)
	if !ok {
		return "shader"
	}
	var parts []string
	for _, part := range strings.Split(stem, "/") {
		parts = append(parts, words(part)...)
	}
	parts = append(parts, stageToken(stage))
	parts = slices.DeleteFunc(parts, func(part string) bool { return part == "" })
	if len(parts) == 0 {
		return "shader"
	}
	var b strings.Builder
	for i, part := range parts {
		lower := strings.ToLower(part)
		if i == 0 {
			b.WriteString(lower)
			continue
		}
		b.WriteString(strings.ToUpper(lower[:1]))
		if len(lower) > 1 {
			b.WriteString(lower[1:])
		}
	}
	name := b.String()
	if name == "" {
		name = "shader"
	}
	if r := rune(name[0]); r >= '0' && r <= '9' {
		name = "shader" + strings.ToUpper(name[:1]) + name[1:]
	}
	if !token.IsIdentifier(name) || token.IsKeyword(name) {
		name = "shader" + strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

func words(part string) []string {
	var cur strings.Builder
	var out []string
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, cur.String())
		cur.Reset()
	}
	for _, r := range part {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func uniqueIdent(name string, used map[string]int) string {
	n := used[name]
	used[name] = n + 1
	if n == 0 {
		return name
	}
	return fmt.Sprintf("%s%d", name, n+1)
}
