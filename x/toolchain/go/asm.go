package gotool

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	asmarch "github.com/lewtec/lewkit/x/toolchain/go/src/asm/arch"
	asmpkg "github.com/lewtec/lewkit/x/toolchain/go/src/asm/asm"
	asmflags "github.com/lewtec/lewkit/x/toolchain/go/src/asm/flags"
	asmlex "github.com/lewtec/lewkit/x/toolchain/go/src/asm/lex"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/bio"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/obj"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/objabi"
	"github.com/lewtec/lewkit/x/toolchain/go/src/std/buildcfg"
)

// asmHere assembles req.Files in this process.
// The caller is the toolchain child, so GOOS and GOARCH are the target.
// One process owns one assembler context and exits with it.
func asmHere(req childRequest) error {
	log.SetFlags(0)
	log.SetPrefix("asm: ")
	buildcfg.Check()
	if req.Out == "" || req.ImportPath == "" || len(req.Files) == 0 {
		return fmt.Errorf("%w: missing output, package, or files", ErrAsm)
	}
	defs, err := asmDefines()
	if err != nil {
		return err
	}
	*asmflags.OutputFile = req.Out
	*asmflags.Importpath = req.ImportPath
	*asmflags.SymABIs = req.GenSymABIs
	*asmflags.Shared = req.Shared
	*asmflags.Std = req.Std
	*asmflags.TrimPath = req.Trimpath
	*asmflags.Dynlink = false
	*asmflags.Linkshared = false
	*asmflags.Spectre = ""
	asmflags.D = defs
	asmflags.I = append([]string(nil), req.Includes...)
	if objabi.LookupPkgSpecial(req.ImportPath).AllowAsmABI {
		for _, exp := range buildcfg.Experiment.Enabled() {
			asmflags.D = append(asmflags.D, "GOEXPERIMENT_"+exp)
		}
	}

	architecture := asmarch.Set(buildcfg.GOARCH, req.Shared)
	if architecture == nil {
		return fmt.Errorf("%w %s", ErrCompileSupportsGoarchAmd64OrArm64Have, buildcfg.GOARCH)
	}
	ctxt := obj.Linknew(architecture.LinkArch)
	ctxt.CompressInstructions = asmflags.DebugFlags.CompressInstructions != 0
	ctxt.IsAsm = true
	ctxt.Pkgpath = req.ImportPath
	ctxt.Std = req.Std
	ctxt.Flag_shared = req.Shared
	ctxt.DwTextCount = objabi.DummyDwarfFunctionCountForAssembler()
	if architecture.Init != nil {
		architecture.Init(ctxt)
	}

	buf, err := bio.Create(req.Out)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	closed := false
	defer func() {
		if !closed {
			buf.Close()
		}
	}()
	if !req.GenSymABIs {
		buf.WriteString(objabi.HeaderString())
		fmt.Fprintf(buf, "!\n")
	}

	failed := ""
	diag := false
	for _, file := range req.Files {
		lexer := asmlex.NewLexer(file)
		parser := asmpkg.NewParser(ctxt, architecture, lexer)
		ctxt.DiagFunc = func(format string, args ...any) {
			diag = true
			log.Printf(format, args...)
		}
		ok := false
		if req.GenSymABIs {
			ok = parser.ParseSymABIs(buf)
		} else {
			plist := new(obj.Plist)
			plist.Firstpc, ok = parser.Parse()
			if ok {
				obj.Flushplist(ctxt, plist, nil)
			}
		}
		if !ok {
			failed = file
			break
		}
	}
	if failed == "" && !diag && !req.GenSymABIs {
		ctxt.NumberSyms()
		obj.WriteObjFile(ctxt, buf)
	}
	if err := buf.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	closed = true
	if failed != "" || diag {
		os.Remove(req.Out)
		if failed != "" {
			return fmt.Errorf("%w: %s", ErrAsm, failed)
		}
		return fmt.Errorf("%w: assembly failed", ErrAsm)
	}
	return nil
}

func asmDefines() ([]string, error) {
	defs := []string{"GOOS_" + buildcfg.GOOS, "GOARCH_" + buildcfg.GOARCH}
	switch buildcfg.GOARCH {
	case "amd64":
		defs = append(defs, fmt.Sprintf("GOAMD64_v%d", buildcfg.GOAMD64))
	case "arm64":
		if buildcfg.GOARM64.LSE {
			defs = append(defs, "GOARM64_LSE")
		}
	default:
		return nil, fmt.Errorf("%w %s", ErrCompileSupportsGoarchAmd64OrArm64Have, buildcfg.GOARCH)
	}
	return defs, nil
}

// buildAssembled compiles a package that has .s or .syso files.
// gensymabis runs first, against an empty go_asm.h. The compiler then
// writes that header. One assembler invocation writes one object, and
// that object plus any system objects are packed into the archive.
func buildAssembled(ctx context.Context, u Unit) error {
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	scratch, err := os.MkdirTemp("", "lewkit-asm-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	part := partPath(u.Archive)
	removePart := true
	defer func() {
		if removePart {
			os.Remove(part)
		}
	}()

	includes := []string{}
	asmhdr := ""
	symabis := ""
	if len(u.SFiles) > 0 {
		hdr := filepath.Join(scratch, "go_asm.h")
		if err := os.WriteFile(hdr, nil, 0o644); err != nil {
			return err
		}
		asmhdr = hdr
		includes = append(includes, scratch)
		if u.Include != "" {
			includes = append(includes, u.Include)
		}
		symabis = filepath.Join(scratch, "symabis")
		if err := spawn(ctx, "asm", asmRequest(u, symabis, u.SFiles, includes, true)); err != nil {
			return err
		}
	}
	if err := spawn(ctx, "compile", childRequest{
		ImportPath: u.ImportPath,
		Out:        part,
		ImportCfg:  u.ImportCfg,
		EmbedCfg:   u.EmbedCfg,
		Lang:       u.Lang,
		Shared:     u.Shared,
		Std:        u.Std,
		NoComplete: true,
		Files:      u.Files,
		GOOS:       u.GOOS,
		GOARCH:     u.GOARCH,
		Trimpath:   u.Trimpath,
		SymABIs:    symabis,
		Asmhdr:     asmhdr,
	}); err != nil {
		return err
	}

	objs := []string{}
	if len(u.SFiles) > 0 {
		obj := filepath.Join(scratch, "asm.o")
		if err := spawn(ctx, "asm", asmRequest(u, obj, u.SFiles, includes, false)); err != nil {
			return err
		}
		objs = append(objs, obj)
	}
	objs = append(objs, u.SysoFiles...)
	if len(objs) > 0 {
		if err := packObjects(part, objs); err != nil {
			return err
		}
	}
	if err := os.Rename(part, u.Archive); err != nil {
		return err
	}
	removePart = false
	return nil
}

func asmRequest(u Unit, out string, files, includes []string, symabis bool) childRequest {
	return childRequest{
		ImportPath: u.ImportPath,
		Out:        out,
		Files:      files,
		GenSymABIs: symabis,
		Includes:   includes,
		Std:        u.Std,
		Shared:     u.Shared,
		Trimpath:   u.Trimpath,
		GOOS:       u.GOOS,
		GOARCH:     u.GOARCH,
	}
}
