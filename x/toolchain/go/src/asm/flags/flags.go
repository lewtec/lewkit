// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package flags implements top-level flags and the usage message for the assembler.
package flags

import (
	"flag"
	"fmt"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/obj"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/objabi"
	"os"
	"path/filepath"
	"strings"
)

var (
	Debug      *bool
	OutputFile *string
	TrimPath   *string
	Shared     *bool
	Dynlink    *bool
	Linkshared *bool
	AllErrors  *bool
	SymABIs    *bool
	Importpath *string
	Spectre    *string
	Std        *bool
)

var DebugFlags struct {
	CompressInstructions int    `help:"use compressed instructions when possible (if supported by architecture)"`
	MayMoreStack         string `help:"call named function before all stack growth checks"`
	PCTab                string `help:"print named pc-value table\nOne of: pctospadj, pctofile, pctoline, pctoinline, pctopcdata"`
}

var (
	D        MultiFlag
	I        MultiFlag
	PrintOut int
	DebugV   bool
)

func init() {
	fs := flag.NewFlagSet("asm", flag.ContinueOnError)
	Debug = fs.Bool("debug", false, "dump instructions as they are parsed")
	OutputFile = fs.String("o", "", "output file; default foo.o for /a/b/c/foo.s as first argument")
	TrimPath = fs.String("trimpath", "", "remove prefix from recorded source file paths")
	Shared = fs.Bool("shared", false, "generate code that can be linked into a shared library")
	Dynlink = fs.Bool("dynlink", false, "support references to Go symbols defined in other shared libraries")
	Linkshared = fs.Bool("linkshared", false, "generate code that will be linked against Go shared libraries")
	AllErrors = fs.Bool("e", false, "no limit on number of errors reported")
	SymABIs = fs.Bool("gensymabis", false, "write symbol ABI information to output file, don't assemble")
	Importpath = fs.String("p", obj.UnlinkablePkg, "set expected package import to path")
	Spectre = fs.String("spectre", "", "enable spectre mitigations in `list` (all, ret)")
	Std = fs.Bool("std", false, "building standard library")
	fs.Var(&D, "D", "predefined symbol with optional simple value -D=identifier=value; can be set multiple times")
	fs.Var(&I, "I", "include directory; can be set multiple times")
	fs.BoolVar(&DebugV, "v", false, "print debug output")
	fs.Var(objabi.NewDebugFlag(&DebugFlags, nil), "d", "enable debugging settings; try -d help")
	objabi.FlagcountFS(fs, "S", "print assembly and machine code", &PrintOut)
	DebugFlags.CompressInstructions = 1
}

// MultiFlag allows setting a value multiple times to collect a list, as in -I=dir1 -I=dir2.
type MultiFlag []string

func (m *MultiFlag) String() string {
	if len(*m) == 0 {
		return ""
	}
	return fmt.Sprint(*m)
}

func (m *MultiFlag) Set(val string) error {
	(*m) = append(*m, val)
	return nil
}

func Usage() {
	fmt.Fprintf(os.Stderr, "usage: asm [options] file.s ...\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func Parse() {
	objabi.Flagparse(Usage)
	if flag.NArg() == 0 {
		flag.Usage()
	}

	// Flag refinement.
	if *OutputFile == "" {
		if flag.NArg() != 1 {
			flag.Usage()
		}
		input := filepath.Base(flag.Arg(0))
		input = strings.TrimSuffix(input, ".s")
		*OutputFile = fmt.Sprintf("%s.o", input)
	}
}
