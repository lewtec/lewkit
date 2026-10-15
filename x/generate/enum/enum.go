// Package enum writes String, Parse, Values, IsValid, and text
// marshaling for types marked in the same file.
//
// A mark is a comment line on the type:
//
//	//lewkit:enum
//	type Format int
//
//	const (
//		FormatText Format = iota + 1
//		FormatTable
//	)
//
// The type is an integer or a string. Members are the const values of
// that type. An integer token is the member name with the type name
// removed, in lower case. A line comment that is only a comma-separated
// token list sets that token and any extra names that parse to it.
// A string member uses its literal, and the comment is only extra names.
//
//	//lewkit:enum fold empty=warning invalid=unknown
//
// fold parses without case. empty=<token> accepts a blank string as
// that member. invalid=<text> is what String returns for a value
// outside the set; otherwise String returns an empty string.
//
// The generated type is a [github.com/lewtec/lewkit/x/cmd.Enum] when it
// is an integer, and a flag either way: Parse reads the text and
// ArgChoices lists the accepted texts for help.
//
// Run writes <name>_enum.go beside each file it is given. With no
// file, it writes one for every marked file in the package directory.
// That directory is the working directory. go generate runs there and
// sets GOFILE to the base name of the file that holds the directive.
// A GOFILE that names a path selects that file's directory.
//
// The directive lives in enum.go. That file is the package clause and
// the directive. It is not an input, so the generator does not write
// enum_enum.go:
//
//	package sample
//
//	//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum
package enum

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/generate"
)

// Run reads each Go file and writes <name>_enum.go beside it.
// With no file, Run reads every marked file in the package directory.
func Run(ctx context.Context, files []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(files) == 0 {
		return runPackage(ctx)
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runFile(file); err != nil {
			return err
		}
	}
	return nil
}

func runPackage(ctx context.Context) error {
	dir, err := packageDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	wrote := false
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() || !isEnumSource(name) {
			continue
		}
		path := filepath.Join(dir, name)
		ok, err := generateFile(path, true)
		if err != nil {
			return err
		}
		wrote = wrote || ok
	}
	if !wrote {
		return fmt.Errorf("%w: %s", ErrNoEnum, dir)
	}
	return nil
}

// packageDir is the working directory. A GOFILE that includes a
// directory selects that directory.
func packageDir() (string, error) {
	gofile := os.Getenv("GOFILE")
	if gofile != "" && (filepath.IsAbs(gofile) || filepath.Dir(gofile) != ".") {
		abs, err := filepath.Abs(gofile)
		if err != nil {
			return "", err
		}
		return filepath.Dir(abs), nil
	}
	return os.Getwd()
}

func runFile(path string) error {
	wrote, err := generateFile(path, false)
	if err != nil {
		return err
	}
	if !wrote {
		return fmt.Errorf("%w: %s", ErrNoEnum, path)
	}
	return nil
}

// generateFile writes <name>_enum.go when path has a marked type.
// skipUnmarked leaves a file with no enum unwritten.
func generateFile(path string, skipUnmarked bool) (bool, error) {
	out, err := outputPath(path)
	if err != nil {
		return false, err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if skipUnmarked && !bytes.Contains(src, []byte("lewkit:enum")) {
		return false, nil
	}
	parsed, err := parseSource(path, src)
	if err != nil {
		return false, err
	}
	if len(parsed.enums) == 0 {
		return false, nil
	}
	buf, err := render(parsed)
	if err != nil {
		return false, err
	}
	if err := generate.WriteFile(out, buf); err != nil {
		return false, err
	}
	return true, nil
}

// isEnumSource reports whether name is a Go file the generator reads.
// enum.go holds the go:generate line and is not an input, so the
// output is never enum_enum.go.
func isEnumSource(name string) bool {
	return strings.HasSuffix(name, ".go") &&
		!strings.HasSuffix(name, "_test.go") &&
		!strings.HasSuffix(name, "_enum.go") &&
		name != "enum.go"
}

func outputPath(path string) (string, error) {
	if !isEnumSource(filepath.Base(path)) {
		return "", fmt.Errorf("%w: %s", ErrEnum, path)
	}
	return strings.TrimSuffix(path, ".go") + "_enum.go", nil
}
