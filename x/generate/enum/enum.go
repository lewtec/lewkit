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
// go generate runs it beside the declaration:
//
//	//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum format.go
package enum

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/generate"
)

// Run reads each Go file and writes <name>_enum.go beside it.
func Run(ctx context.Context, files []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(files) == 0 {
		return ErrFileRequired
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

func runFile(path string) error {
	out, err := outputPath(path)
	if err != nil {
		return err
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	parsed, err := parseSource(path, src)
	if err != nil {
		return err
	}
	if len(parsed.enums) == 0 {
		return fmt.Errorf("%w: %s", ErrNoEnum, path)
	}
	buf, err := render(parsed)
	if err != nil {
		return err
	}
	return generate.WriteFile(out, buf)
}

func outputPath(path string) (string, error) {
	if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_enum.go") {
		return "", fmt.Errorf("%w: %s", ErrEnum, path)
	}
	return strings.TrimSuffix(path, ".go") + "_enum.go", nil
}
