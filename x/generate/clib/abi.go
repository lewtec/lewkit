package clib

import (
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/generate/header"
)

type machine int

const (
	machAmd64 machine = iota
	machArm64
	mach386
	machARM
	machWin64
	machWin386
)

type locKind int

const (
	locGPR locKind = iota
	locXMM
	locStack
	locArmF32
	locArmF64
)

// loc is where one C argument lives.
// a is the integer slot, the xmm slot, or the cdecl byte offset.
// b is the high arm register of a 64-bit integer, or -1.
// s is the AAPCS single-precision register.
type loc struct {
	kind locKind
	a    int
	b    int
	s    int
}

type target struct {
	goos       string
	goarch     string
	mach       machine
	constraint string
	// fileOS is the GOOS in the generated file name. Empty means the
	// file is shared by linux and android, so the name carries only the arch.
	fileOS string
}

func (t target) header() (header.Target, error) {
	return header.For(t.goos, t.goarch)
}

func linuxTargets() []target {
	return []target{
		{goos: "linux", goarch: "amd64", mach: machAmd64, constraint: "(linux || android) && amd64"},
		{goos: "linux", goarch: "arm64", mach: machArm64, constraint: "(linux || android) && arm64"},
		{goos: "linux", goarch: "386", mach: mach386, constraint: "(linux || android) && 386"},
		{goos: "linux", goarch: "arm", mach: machARM, constraint: "(linux || android) && arm"},
	}
}

func windowsTargets() []target {
	return []target{
		{goos: "windows", goarch: "amd64", mach: machWin64, constraint: "windows && amd64", fileOS: "windows"},
		{goos: "windows", goarch: "arm64", mach: machArm64, constraint: "windows && arm64", fileOS: "windows"},
		{goos: "windows", goarch: "386", mach: machWin386, constraint: "windows && 386", fileOS: "windows"},
	}
}

func darwinTargets() []target {
	return []target{
		{goos: "darwin", goarch: "amd64", mach: machAmd64, constraint: "darwin && amd64", fileOS: "darwin"},
		{goos: "darwin", goarch: "arm64", mach: machArm64, constraint: "darwin && arm64", fileOS: "darwin"},
	}
}

func targetsFor(spec string) ([]target, error) {
	switch spec {
	case "", "all":
		return linuxTargets(), nil
	case "windows":
		return windowsTargets(), nil
	case "darwin":
		return darwinTargets(), nil
	case "host":
		spec = runtime.GOOS + "/" + runtime.GOARCH
	case "sysv64":
		spec = "linux/amd64"
	case "386":
		spec = "linux/386"
	case "arm":
		spec = "linux/arm"
	case "win64":
		spec = "windows/amd64"
	case "win386":
		spec = "windows/386"
	case "darwinarm64":
		spec = "darwin/arm64"
	}
	parsed, err := header.ParseABI(spec)
	if err != nil {
		return nil, err
	}
	if parsed.GOOS == "" || parsed.GOARCH == "" {
		return nil, fmt.Errorf("%w: %s", ErrHost, spec)
	}
	var list []target
	switch parsed.GOOS {
	case "linux", "android":
		list = linuxTargets()
	case "windows":
		list = windowsTargets()
	case "darwin":
		list = darwinTargets()
	default:
		return nil, fmt.Errorf("%w: %s/%s", ErrHost, parsed.GOOS, parsed.GOARCH)
	}
	for _, t := range list {
		if t.goarch == parsed.GOARCH {
			return []target{t}, nil
		}
	}
	return nil, fmt.Errorf("%w: %s/%s", ErrHost, parsed.GOOS, parsed.GOARCH)
}

func place(m machine, params []param) error {
	switch m {
	case machAmd64:
		return placeSplit(params, 6, 8)
	case machArm64:
		return placeSplit(params, 8, 8)
	case machWin64:
		return placeWin64(params)
	case mach386:
		return placeStack(params, 4)
	case machWin386:
		return placeStack(params, 8)
	default:
		return placeARM(params)
	}
}

// placeWin64 gives every argument one position. The first four use a
// register or an xmm register. The rest are 8-byte stack slots. The
// entry copies each stack slot into both images so the handler can read
// the one that matches the C type.
func placeWin64(params []param) error {
	for i := range params {
		if i >= 8 {
			return ErrType
		}
		p := &params[i]
		kind := locGPR
		if p.val.reg == regXMM {
			kind = locXMM
		}
		p.at = loc{kind: kind, a: i, b: -1}
	}
	return nil
}

func placeSplit(params []param, gprMax, xmmMax int) error {
	gpr, xmm := 0, 0
	for i := range params {
		p := &params[i]
		if p.val.reg == regXMM {
			if xmm >= xmmMax {
				return ErrType
			}
			p.at = loc{kind: locXMM, a: xmm, b: -1}
			xmm++
			continue
		}
		if gpr >= gprMax {
			return ErrType
		}
		p.at = loc{kind: locGPR, a: gpr, b: -1}
		gpr++
	}
	return nil
}

// placeStack packs cdecl arguments. wideAlign is 4 on Unix 386 and 8 on
// Win386, where a long long or a double is aligned to 8 and the compiler
// inserts the pad.
func placeStack(params []param, wideAlign int) error {
	off := 0
	for i := range params {
		p := &params[i]
		stride := p.val.size
		if stride < 4 {
			stride = 4
		}
		if stride != 4 && stride != 8 {
			return ErrType
		}
		if stride == 8 && wideAlign > 4 {
			off = (off + wideAlign - 1) &^ (wideAlign - 1)
		}
		if off+stride > 64 {
			return ErrType
		}
		p.at = loc{kind: locStack, a: off, b: -1}
		off += stride
	}
	return nil
}

// placeARM follows the AAPCS hard-float rules. A 64-bit integer uses an
// even-odd register pair. A double uses an even single-precision pair,
// and a later float may fill the single that was skipped.
func placeARM(params []param) error {
	gpr := 0
	nsrn := 0
	var holes []int
	for i := range params {
		p := &params[i]
		if p.val.reg == regXMM {
			s, next, left, err := armFloat(p.val.size == 8, nsrn, holes)
			if err != nil {
				return err
			}
			nsrn, holes = next, left
			kind := locArmF32
			if p.val.size == 8 {
				kind = locArmF64
			}
			p.at = loc{kind: kind, a: s / 2, b: -1, s: s}
			continue
		}
		words := 1
		if p.val.size > 4 {
			if p.val.size != 8 {
				return ErrType
			}
			words = 2
		}
		if words == 2 && gpr%2 == 1 {
			gpr++
		}
		if gpr+words > 4 {
			return ErrType
		}
		hi := -1
		if words == 2 {
			hi = gpr + 1
		}
		p.at = loc{kind: locGPR, a: gpr, b: hi}
		gpr += words
	}
	return nil
}

func armFloat(double bool, nsrn int, holes []int) (s, next int, left []int, err error) {
	if !double {
		if len(holes) > 0 {
			return holes[0], nsrn, holes[1:], nil
		}
		if nsrn >= 16 {
			return 0, 0, nil, ErrType
		}
		return nsrn, nsrn + 1, holes, nil
	}
	if nsrn%2 == 1 {
		holes = append(holes, nsrn)
		nsrn++
	}
	if nsrn+1 >= 16 {
		return 0, 0, nil, ErrType
	}
	return nsrn, nsrn + 2, holes, nil
}
