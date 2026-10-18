// Package parse writes the LR(1) table for one language package.
//
// The package exports ParseTable. Run calls that function and writes
// table_gen.go beside it. The variable is the package name plus Table.
// go generate runs in the package directory and sets GOFILE to the file
// that holds the directive. A GOFILE that names a path selects that
// file's directory.
//
//	//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse
package parse

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

const (
	tableFile  = "table_gen.go"
	emitFile   = "parse_table_main.go"
	tableMagic = "LK01"
)

// Run writes table_gen.go for the language package in the working directory.
func Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := packageDir()
	if err != nil {
		return err
	}
	name, importPath, err := listPackage(ctx, dir)
	if err != nil {
		return err
	}
	table, err := compileTable(ctx, dir, importPath)
	if err != nil {
		return err
	}
	body, err := format(name, name+"Table", table)
	if err != nil {
		return err
	}
	return generate.WriteFile(filepath.Join(dir, tableFile), body)
}

func format(pkg, name string, table []byte) ([]byte, error) {
	file := golang.NewFile(pkg)
	file.Header = "// " + generate.Notice("parse")
	file.Var().Id(name).Values(golang.Slice(golang.Named("byte")).AsExpr().Call(golang.Lit(string(table))))
	var buf bytes.Buffer
	if err := file.Make(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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

func listPackage(ctx context.Context, dir string) (string, string, error) {
	stdout, stderr, err := runGo(ctx, dir, "list", "-f", "{{.Name}}\t{{.ImportPath}}")
	if err != nil {
		return "", "", fmt.Errorf("%w: %s", ErrNoPackage, commandText(stdout, stderr, err))
	}
	fields := strings.Split(strings.TrimSpace(string(stdout)), "\t")
	if len(fields) != 2 || fields[0] == "" || fields[1] == "" {
		return "", "", fmt.Errorf("%w: %s", ErrNoPackage, strings.TrimSpace(string(stdout)))
	}
	return fields[0], fields[1], nil
}

// compileTable runs ParseTable inside the language package.
// The declaration is ordinary Go, so this writes a throwaway main
// next to it and reads the table from that program's stdout.
// The build tag keeps the main out of the language package.
func compileTable(ctx context.Context, dir, importPath string) ([]byte, error) {
	path := filepath.Join(dir, emitFile)
	source, err := emitSource(importPath)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, source, 0o644); err != nil {
		return nil, err
	}
	defer os.Remove(path)
	stdout, stderr, err := runGo(ctx, dir, "run", emitFile)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("%w: %s", ErrTable, commandText(stdout, stderr, err))
	}
	if !bytes.HasPrefix(stdout, []byte(tableMagic)) {
		return nil, fmt.Errorf("%w: table does not start with %s", ErrTable, tableMagic)
	}
	return stdout, nil
}

func emitSource(importPath string) ([]byte, error) {
	file := golang.NewFile("main")
	file.Header = "//go:build ignore"
	file.Import("os")
	grammar := file.Import(importPath).Alias("grammar")
	grammar.Group = 1
	file.Func().Id("main").Block(
		golang.Define(
			[]*golang.Expr{golang.Id("table"), golang.Id("err")},
			golang.Qual(importPath, "ParseTable").Call(),
		),
		exitOnError(),
		exitOnError().WithInit(golang.Define(
			[]*golang.Expr{golang.Id("_"), golang.Id("err")},
			golang.Qual("os", "Stdout").Sel("Write").Call(golang.Id("table")),
		)),
	)
	var buf bytes.Buffer
	if err := file.Make(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// exitOnError writes err to stderr and stops the process.
func exitOnError() *golang.Stmt {
	return golang.If(
		golang.Binary(golang.OpNeq, golang.Id("err"), golang.Lit(nil)),
		golang.Qual("os", "Stderr").Sel("WriteString").Call(
			golang.Binary(golang.OpAdd, golang.Id("err").Sel("Error").Call(), golang.Lit("\n")),
		),
		golang.Qual("os", "Exit").Call(golang.Lit(1)),
	)
}

func runGo(ctx context.Context, dir string, args ...string) ([]byte, []byte, error) {
	cmd := execdriver.MustCommand(ctx, "go", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := execdriver.Run(ctx, cmd)
	return stdout.Bytes(), stderr.Bytes(), err
}

func commandText(stdout, stderr []byte, err error) string {
	text := strings.TrimSpace(string(stderr))
	if text == "" {
		text = strings.TrimSpace(string(stdout))
	}
	if text == "" {
		text = err.Error()
	}
	return text
}
