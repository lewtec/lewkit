// Inferno utils/6l/obj.c
// https://bitbucket.org/inferno-os/inferno-os/src/master/utils/6l/obj.c
//
//	Copyright © 1994-1999 Lucent Technologies Inc.  All rights reserved.
//	Portions Copyright © 1995-1997 C H Forsyth (forsyth@terzarima.net)
//	Portions Copyright © 1997-1999 Vita Nuova Limited
//	Portions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com)
//	Portions Copyright © 2004,2006 Bruce Ellis
//	Portions Copyright © 2005-2007 C H Forsyth (forsyth@terzarima.net)
//	Revisions Copyright © 2000-2007 Lucent Technologies Inc. and others
//	Portions Copyright © 2009 The Go Authors. All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package ld

import (
	"bufio"
	"bytes"
	"flag"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/goobj"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/objabi"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/quoted"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/sys"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/telemetry/counter"
	"github.com/lewtec/lewkit/x/toolchain/go/src/link/benchmark"
	"github.com/lewtec/lewkit/x/toolchain/go/src/std/buildcfg"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
)

var (
	pkglistfornote []byte
	windowsgui     bool // writes a "GUI binary" instead of a "console binary"
	ownTmpDir      bool // set to true if tmp dir created by linker (e.g. no -tmpdir)
)

// Flags used by the linker. The exported flags are used by the architecture-specific packages.
// bindFlags fills them on a private FlagSet. Importing this package does not
// register anything on flag.CommandLine.
var (
	flagBuildid *string
	flagBindNow *bool

	flagOutfile    *string
	flagPluginPath *string
	flagFipso      *string

	flagInstallSuffix *string
	flagDumpDep       *bool
	flagRace          *bool
	flagMsan          *bool
	flagAsan          *bool
	flagAslr          *bool

	flagFieldTrack *string
	flagLibGCC     *string
	flagTmpdir     *string

	flagExtld      quoted.Flag
	flagExtldflags quoted.Flag
	flagExtar      *string

	flagCaptureHostObjs *string

	flagA             *bool
	FlagC             *bool
	FlagD             *bool
	flagF             *bool
	flagG             *bool
	flagH             *bool
	flagN             *bool
	FlagS             *bool
	flag8             bool // use 64-bit addresses in symbol table
	flagHostBuildid   *string
	flagInterpreter   *string
	flagCheckLinkname *bool
	FlagDebugTramp    *int
	FlagDebugTextSize *int
	flagDebugNosplit  *bool
	FlagStrictDups    *int
	FlagRound         *int64
	FlagTextAddr      *int64
	FlagDataAddr      *int64
	FlagFuncAlign     *int
	flagEntrySymbol   *string
	flagPruneWeakMap  *bool
	flagRandLayout    *int64
	flagAllErrors     *bool
	cpuprofile        *string
	memprofile        *string
	memprofilerate    *int64
	benchmarkFlag     *string
	benchmarkFileFlag *string

	flagW ternaryFlag
	FlagW *bool // the -w flag, computed in main from flagW
)

// bindFlags registers linker flags on fs and returns the -H value.
func bindFlags(fs *flag.FlagSet, ctxt *Link) *string {
	flagW = ternaryFlagUnset
	flag8 = false
	flagExtld = quoted.Flag{}
	flagExtldflags = quoted.Flag{}
	FlagW = new(bool)

	flagBuildid = fs.String("buildid", "", "record `id` as Go toolchain build id")
	flagBindNow = fs.Bool("bindnow", false, "mark a dynamically linked ELF object for immediate function binding")
	flagOutfile = fs.String("o", "", "write output to `file`")
	flagPluginPath = fs.String("pluginpath", "", "full path name for plugin")
	flagFipso = fs.String("fipso", "", "write fips module to `file`")
	flagInstallSuffix = fs.String("installsuffix", "", "set package directory `suffix`")
	flagDumpDep = fs.Bool("dumpdep", false, "dump symbol dependency graph")
	flagRace = fs.Bool("race", false, "enable race detector")
	flagMsan = fs.Bool("msan", false, "enable MSan interface")
	flagAsan = fs.Bool("asan", false, "enable ASan interface")
	flagAslr = fs.Bool("aslr", true, "enable ASLR for buildmode=c-shared on windows")
	flagFieldTrack = fs.String("k", "", "set field tracking `symbol`")
	flagLibGCC = fs.String("libgcc", "", "compiler support lib for internal linking; use \"none\" to disable")
	flagTmpdir = fs.String("tmpdir", "", "use `directory` for temporary files")
	fs.Var(&flagExtld, "extld", "use `linker` when linking in external mode")
	fs.Var(&flagExtldflags, "extldflags", "pass `flags` to external linker")
	flagExtar = fs.String("extar", "", "archive program for buildmode=c-archive")
	flagCaptureHostObjs = fs.String("capturehostobjs", "", "capture host object files loaded during internal linking to specified dir")
	flagA = fs.Bool("a", false, "no-op (deprecated)")
	FlagC = fs.Bool("c", false, "dump call graph")
	FlagD = fs.Bool("d", false, "disable dynamic executable")
	flagF = fs.Bool("f", false, "ignore version mismatch")
	flagG = fs.Bool("g", false, "disable go package data checks")
	flagH = fs.Bool("h", false, "halt on error")
	flagN = fs.Bool("n", false, "no-op (deprecated)")
	FlagS = fs.Bool("s", false, "disable symbol table")
	flagHostBuildid = fs.String("B", "", "set ELF NT_GNU_BUILD_ID `note` or Mach-O UUID; use \"gobuildid\" to generate it from the Go build ID; \"none\" to disable")
	flagInterpreter = fs.String("I", "", "use `linker` as ELF dynamic linker")
	flagCheckLinkname = fs.Bool("checklinkname", true, "check linkname symbol references")
	FlagDebugTramp = fs.Int("debugtramp", 0, "debug trampolines")
	FlagDebugTextSize = fs.Int("debugtextsize", 0, "debug text section max size")
	flagDebugNosplit = fs.Bool("debugnosplit", false, "dump nosplit call graph")
	FlagStrictDups = fs.Int("strictdups", 0, "sanity check duplicate symbol contents during object file reading (1=warn 2=err).")
	FlagRound = fs.Int64("R", -1, "set address rounding `quantum`")
	FlagTextAddr = fs.Int64("T", -1, "set the start address of text symbols")
	FlagDataAddr = fs.Int64("D", -1, "set the start address of data symbols")
	FlagFuncAlign = fs.Int("funcalign", 0, "set function align to `N` bytes")
	flagEntrySymbol = fs.String("E", "", "set `entry` symbol name")
	flagPruneWeakMap = fs.Bool("pruneweakmap", true, "prune weak mapinit refs")
	flagRandLayout = fs.Int64("randlayout", 0, "randomize function layout")
	flagAllErrors = fs.Bool("e", false, "no limit on number of errors reported")
	cpuprofile = fs.String("cpuprofile", "", "write cpu profile to `file`")
	memprofile = fs.String("memprofile", "", "write memory profile to `file`")
	memprofilerate = fs.Int64("memprofilerate", 0, "set runtime.MemProfileRate to `rate`")
	benchmarkFlag = fs.String("benchmark", "", "set to 'mem' or 'cpu' to enable phase benchmarking")
	benchmarkFileFlag = fs.String("benchmarkprofile", "", "emit phase profiles to `base`_phase.{cpu,mem}prof")
	fs.Var(&macOS, "macos", "mac OS version to write in build info (only used in internal linking)")
	fs.Var(&macSDK, "macsdk", "mac SDK version to write in build info (only used in internal linking)")
	fs.Var(&rpath, "r", "set the ELF dynamic linker search `path` to dir1:dir2:...")
	fs.Var(&flagW, "w", "disable DWARF generation")

	flagHeadType := fs.String("H", "", "set header `type`")
	fs.BoolVar(&ctxt.linkShared, "linkshared", false, "link against installed Go shared libraries")
	fs.Var(&ctxt.LinkMode, "linkmode", "set link `mode`")
	fs.Var(&ctxt.BuildMode, "buildmode", "set build `mode`")
	fs.BoolVar(&ctxt.compressDWARF, "compressdwarf", true, "compress DWARF if possible")
	if ctxt.Arch.Family == sys.AMD64 && buildcfg.GOOS == "plan9" {
		fs.BoolVar(&flag8, "8", false, "use 64-bit addresses in symbol table")
	}
	objabi.Flagfn1FS(fs, "L", "add specified `directory` to library path", func(a string) { Lflag(ctxt, a) })
	objabi.AddVersionFlagFS(fs)
	objabi.Flagfn1FS(fs, "X", "add string value `definition` of the form importpath.name=value", func(s string) { addstrdata1(ctxt, s) })
	objabi.FlagcountFS(fs, "v", "print link trace", &ctxt.Debugvlog)
	objabi.Flagfn1FS(fs, "importcfg", "read import configuration from `file`", ctxt.readImportCfg)
	return flagHeadType
}

// ternaryFlag is like a boolean flag, but has a default value that is
// neither true nor false, allowing it to be set from context (e.g. from another
// flag).
// *ternaryFlag implements flag.Value.
type ternaryFlag int

const (
	ternaryFlagUnset ternaryFlag = iota
	ternaryFlagFalse
	ternaryFlagTrue
)

func (t *ternaryFlag) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	if v {
		*t = ternaryFlagTrue
	} else {
		*t = ternaryFlagFalse
	}
	return nil
}

func (t *ternaryFlag) String() string {
	switch *t {
	case ternaryFlagFalse:
		return "false"
	case ternaryFlagTrue:
		return "true"
	}
	return "unset"
}

func (t *ternaryFlag) IsBoolFlag() bool { return true } // parse like a boolean flag

// Main links the archive named by os.Args.
func Main(arch *sys.Arch, theArch Arch) {
	Run(arch, theArch, os.Args[1:])
}

// Run links the archive named by args. args does not include the program name.
// It does not read os.Args or flag.CommandLine.
func Run(arch *sys.Arch, theArch Arch, args []string) {
	counter.Open()
	counter.Inc("link/invocations")

	thearch = theArch
	ctxt := linknew(arch)
	ctxt.Bso = bufio.NewWriter(diagOut())

	// For testing behavior of go command when tools crash silently.
	// Undocumented, not in standard flag parser to avoid
	// exposing in usage message.
	for _, arg := range args {
		if arg == "-crash_for_testing" {
			Exit(2)
		}
	}

	if buildcfg.GOROOT == "" {
		// cmd/go clears the GOROOT variable when -trimpath is set,
		// so omit it from the binary even if cmd/link itself has an
		// embedded GOROOT value reported by runtime.GOROOT.
	} else {
		addstrdata1(ctxt, "runtime.defaultGOROOT="+buildcfg.GOROOT)
	}

	buildVersion := buildcfg.Version
	if goexperiment := buildcfg.Experiment.String(); goexperiment != "" {
		sep := " "
		if !strings.Contains(buildVersion, "-") { // See go.dev/issue/75953.
			sep = "-"
		}
		buildVersion += sep + "X:" + goexperiment
	}
	addstrdata1(ctxt, "runtime.buildVersion="+buildVersion)

	fs := flag.NewFlagSet("link", flag.ContinueOnError)
	var flagErr bytes.Buffer
	fs.SetOutput(&flagErr)
	fs.Usage = func() {}
	flagHeadType := bindFlags(fs, ctxt)
	if err := objabi.ParseFS(fs, args); err != nil {
		msg := err.Error()
		if text := strings.TrimSpace(flagErr.String()); text != "" {
			msg += "\n" + text
		}
		Exitf("%s", msg)
	}
	files := fs.Args()
	counter.CountFlags("link/flag:", *fs)

	if ctxt.Debugvlog > 0 {
		// dump symbol info on crash
		defer func() { ctxt.loader.Dump() }()
	}
	if ctxt.Debugvlog > 1 {
		// dump symbol info on error
		AtExit(func() {
			if nerrors > 0 {
				ctxt.loader.Dump()
			}
		})
	}

	switch *flagHeadType {
	case "":
	case "windowsgui":
		ctxt.HeadType = objabi.Hwindows
		windowsgui = true
	default:
		if err := ctxt.HeadType.Set(*flagHeadType); err != nil {
			Errorf("%v", err)
			usage()
		}
	}
	if ctxt.HeadType == objabi.Hunknown {
		ctxt.HeadType.Set(buildcfg.GOOS)
	}

	if !*flagAslr && ctxt.BuildMode != BuildModeCShared {
		Errorf("-aslr=false is only allowed for -buildmode=c-shared")
		usage()
	}

	if *FlagD && ctxt.UsesLibc() {
		Exitf("dynamic linking required on %s; -d flag cannot be used", buildcfg.GOOS)
	}

	isPowerOfTwo := func(n int64) bool {
		return n > 0 && n&(n-1) == 0
	}
	if *FlagRound != -1 && (*FlagRound < 4096 || !isPowerOfTwo(*FlagRound)) {
		Exitf("invalid -R value 0x%x", *FlagRound)
	}
	if *FlagFuncAlign != 0 && !isPowerOfTwo(int64(*FlagFuncAlign)) {
		Exitf("invalid -funcalign value %d", *FlagFuncAlign)
	}

	checkStrictDups = *FlagStrictDups

	switch flagW {
	case ternaryFlagFalse:
		*FlagW = false
	case ternaryFlagTrue:
		*FlagW = true
	case ternaryFlagUnset:
		*FlagW = *FlagS // -s implies -w if not explicitly set
		if ctxt.IsDarwin() && ctxt.BuildMode == BuildModeCShared {
			*FlagW = true // default to -w in c-shared mode on darwin, see #61229
		}
	}

	if !buildcfg.Experiment.RegabiWrappers {
		abiInternalVer = 0
	}

	startProfile()
	if ctxt.BuildMode == BuildModeUnset {
		ctxt.BuildMode.Set("exe")
	}

	if ctxt.BuildMode != BuildModeShared && len(files) != 1 {
		usage()
	}

	if *flagOutfile == "" {
		*flagOutfile = "a.out"
		if ctxt.HeadType == objabi.Hwindows {
			*flagOutfile += ".exe"
		}
	}

	interpreter = *flagInterpreter

	if *flagHostBuildid == "" && *flagBuildid != "" {
		*flagHostBuildid = "gobuildid"
	}
	addbuildinfo(ctxt)

	// enable benchmarking
	var bench *benchmark.Metrics
	if len(*benchmarkFlag) != 0 {
		if *benchmarkFlag == "mem" {
			bench = benchmark.New(benchmark.GC, *benchmarkFileFlag)
		} else if *benchmarkFlag == "cpu" {
			bench = benchmark.New(benchmark.NoGC, *benchmarkFileFlag)
		} else {
			Errorf("unknown benchmark flag: %q", *benchmarkFlag)
			usage()
		}
	}

	bench.Start("libinit")
	libinit(ctxt) // creates outfile
	bench.Start("computeTLSOffset")
	ctxt.computeTLSOffset()
	bench.Start("Archinit")
	thearch.Archinit(ctxt)

	if *FlagDataAddr != -1 && *FlagDataAddr%*FlagRound != 0 {
		Exitf("invalid -D value 0x%x: not aligned to rounding quantum 0x%x", *FlagDataAddr, *FlagRound)
	}

	if ctxt.linkShared && !ctxt.IsELF {
		Exitf("-linkshared can only be used on elf systems")
	}

	if ctxt.Debugvlog != 0 {
		onOff := func(b bool) string {
			if b {
				return "on"
			}
			return "off"
		}
		ctxt.Logf("build mode: %s, symbol table: %s, DWARF: %s\n", ctxt.BuildMode, onOff(!*FlagS), onOff(dwarfEnabled(ctxt)))
		ctxt.Logf("HEADER = -H%d -T0x%x -R0x%x\n", ctxt.HeadType, uint64(*FlagTextAddr), uint32(*FlagRound))
	}

	zerofp := goobj.FingerprintType{}
	switch ctxt.BuildMode {
	case BuildModeShared:
		for i := 0; i < len(files); i++ {
			arg := files[i]
			parts := strings.SplitN(arg, "=", 2)
			var pkgpath, file string
			if len(parts) == 1 {
				pkgpath, file = "main", arg
			} else {
				pkgpath, file = parts[0], parts[1]
			}
			pkglistfornote = append(pkglistfornote, pkgpath...)
			pkglistfornote = append(pkglistfornote, '\n')
			addlibpath(ctxt, "command line", "command line", file, pkgpath, "", zerofp)
		}
	case BuildModePlugin:
		addlibpath(ctxt, "command line", "command line", files[0], *flagPluginPath, "", zerofp)
	default:
		addlibpath(ctxt, "command line", "command line", files[0], "main", "", zerofp)
	}
	bench.Start("loadlib")
	ctxt.loadlib()

	bench.Start("inittasks")
	ctxt.inittasks()

	bench.Start("deadcode")
	deadcode(ctxt)

	bench.Start("linksetup")
	ctxt.linksetup()

	bench.Start("dostrdata")
	ctxt.dostrdata()
	if buildcfg.Experiment.FieldTrack {
		bench.Start("fieldtrack")
		fieldtrack(ctxt.Arch, ctxt.loader)
	}

	bench.Start("dwarfGenerateDebugInfo")
	dwarfGenerateDebugInfo(ctxt)

	bench.Start("callgraph")
	ctxt.callgraph()

	bench.Start("doStackCheck")
	ctxt.doStackCheck()

	bench.Start("mangleTypeSym")
	ctxt.mangleTypeSym()

	if ctxt.IsELF {
		bench.Start("doelf")
		ctxt.doelf()
	}
	if ctxt.IsDarwin() {
		bench.Start("domacho")
		ctxt.domacho()
	}
	if ctxt.IsWindows() {
		bench.Start("dope")
		ctxt.dope()
		bench.Start("windynrelocsyms")
		ctxt.windynrelocsyms()
	}
	if ctxt.IsAIX() {
		bench.Start("doxcoff")
		ctxt.doxcoff()
	}

	bench.Start("textbuildid")
	ctxt.textbuildid()
	bench.Start("addexport")
	ctxt.setArchSyms()
	ctxt.addexport()
	bench.Start("Gentext")
	thearch.Gentext(ctxt, ctxt.loader) // trampolines, call stubs, etc.

	bench.Start("textaddress")
	ctxt.textaddress()
	bench.Start("buildinfo")
	ctxt.buildinfo()
	bench.Start("pclntab")
	containers := ctxt.findContainerSyms()
	pclnState := ctxt.pclntab(containers)
	bench.Start("findfunctab")
	ctxt.findfunctab(pclnState, containers)
	bench.Start("dwarfGenerateDebugSyms")
	dwarfGenerateDebugSyms(ctxt)
	bench.Start("symtab")
	symGroupType := ctxt.symtab(pclnState)
	bench.Start("dodata")
	ctxt.dodata(symGroupType)
	bench.Start("address")
	order := ctxt.address()
	bench.Start("dwarfcompress")
	dwarfcompress(ctxt)
	bench.Start("layout")
	filesize := ctxt.layout(order)

	// Write out the output file.
	// It is split into two parts (Asmb and Asmb2). The first
	// part writes most of the content (sections and segments),
	// for which we have computed the size and offset, in a
	// mmap'd region. The second part writes more content, for
	// which we don't know the size.
	if ctxt.Arch.Family != sys.Wasm {
		// Don't mmap if we're building for Wasm. Wasm file
		// layout is very different so filesize is meaningless.
		if err := ctxt.Out.Mmap(filesize); err != nil {
			Exitf("mapping output file failed: %v", err)
		}
	}
	// asmb will redirect symbols to the output file mmap, and relocations
	// will be applied directly there.
	bench.Start("Asmb")
	asmb(ctxt)
	exitIfErrors()

	// Generate additional symbols for the native symbol table just prior
	// to code generation.
	bench.Start("GenSymsLate")
	if thearch.GenSymsLate != nil {
		thearch.GenSymsLate(ctxt, ctxt.loader)
	}

	asmbfips(ctxt, *flagFipso)

	bench.Start("Asmb2")
	asmb2(ctxt)

	bench.Start("Munmap")
	ctxt.Out.Close() // Close handles Munmapping if necessary.

	bench.Start("hostlink")
	ctxt.hostlink()
	if ctxt.Debugvlog != 0 {
		ctxt.Logf("%s", ctxt.loader.Stat())
		ctxt.Logf("%d liveness data\n", liveness)
	}
	bench.Start("Flush")
	ctxt.Bso.Flush()
	bench.Start("archive")
	ctxt.archive()
	bench.Report(os.Stdout)

	errorexit()
}

type Rpath struct {
	set bool
	val string
}

func (r *Rpath) Set(val string) error {
	r.set = true
	r.val = val
	return nil
}

func (r *Rpath) String() string {
	return r.val
}

func startProfile() {
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("%v", err)
		}
		AtExit(func() {
			pprof.StopCPUProfile()
			if err = f.Close(); err != nil {
				log.Fatalf("error closing cpu profile: %v", err)
			}
		})
	}
	if *memprofile != "" {
		if *memprofilerate != 0 {
			runtime.MemProfileRate = int(*memprofilerate)
		}
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatalf("%v", err)
		}
		AtExit(func() {
			// Profile all outstanding allocations.
			runtime.GC()
			// compilebench parses the memory profile to extract memstats,
			// which are only written in the legacy pprof format.
			// See golang.org/issue/18641 and runtime/pprof/pprof.go:writeHeap.
			const writeLegacyFormat = 1
			if err := pprof.Lookup("heap").WriteTo(f, writeLegacyFormat); err != nil {
				log.Fatalf("%v", err)
			}
			// Close the file after writing the profile.
			if err := f.Close(); err != nil {
				log.Fatalf("could not close %v: %v", *memprofile, err)
			}
		})
	}
}
