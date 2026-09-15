package disasm

import (
	"fmt"
	"strings"
)

// Architecture is a Capstone architecture.
type Architecture uint32

const (
	ArchitectureARM        Architecture = 0
	ArchitectureAArch64    Architecture = 1
	ArchitectureMIPS       Architecture = 2
	ArchitectureX86        Architecture = 3
	ArchitecturePowerPC    Architecture = 4
	ArchitectureSPARC      Architecture = 5
	ArchitectureSystemZ    Architecture = 6
	ArchitectureXCore      Architecture = 7
	ArchitectureM68K       Architecture = 8
	ArchitectureTMS320C64X Architecture = 9
	ArchitectureM680X      Architecture = 10
	ArchitectureEVM        Architecture = 11
	ArchitectureMOS65XX    Architecture = 12
	ArchitectureWASM       Architecture = 13
	ArchitectureBPF        Architecture = 14
	ArchitectureRISCV      Architecture = 15
	ArchitectureSH         Architecture = 16
	ArchitectureTriCore    Architecture = 17
	ArchitectureAlpha      Architecture = 18
)

// Mode is a Capstone mode bitset.
type Mode uint32

const (
	ModeLittleEndian Mode = 0
	ModeARM          Mode = 0
	Mode16           Mode = 1 << 1
	Mode32           Mode = 1 << 2
	Mode64           Mode = 1 << 3
	ModeThumb        Mode = 1 << 4
	ModeMClass       Mode = 1 << 5
	ModeV8           Mode = 1 << 6
	ModeMicro        Mode = 1 << 4
	ModeMIPS3        Mode = 1 << 5
	ModeMIPS32R6     Mode = 1 << 6
	ModeMIPS2        Mode = 1 << 7
	ModeV9           Mode = 1 << 4
	ModeBigEndian    Mode = 1 << 31
	ModeMIPS32       Mode = Mode32
	ModeMIPS64       Mode = Mode64
	ModeBPFClassic   Mode = 0
	ModeBPFExtended  Mode = 1 << 0
	ModeRISCV32      Mode = 1 << 0
	ModeRISCV64      Mode = 1 << 1
	ModeRISCVC       Mode = 1 << 2
)

// Syntax is a Capstone assembly syntax.
type Syntax uint32

const (
	SyntaxDefault Syntax = 1 << 1
	SyntaxIntel   Syntax = 1 << 2
	SyntaxATT     Syntax = 1 << 3
	SyntaxNoReg   Syntax = 1 << 4
	SyntaxMASM    Syntax = 1 << 5
)

var architectureNames = map[string]Architecture{
	"arm":        ArchitectureARM,
	"arm64":      ArchitectureAArch64,
	"aarch64":    ArchitectureAArch64,
	"mips":       ArchitectureMIPS,
	"x86":        ArchitectureX86,
	"x86_64":     ArchitectureX86,
	"amd64":      ArchitectureX86,
	"ppc":        ArchitecturePowerPC,
	"powerpc":    ArchitecturePowerPC,
	"sparc":      ArchitectureSPARC,
	"sysz":       ArchitectureSystemZ,
	"systemz":    ArchitectureSystemZ,
	"s390":       ArchitectureSystemZ,
	"xcore":      ArchitectureXCore,
	"m68k":       ArchitectureM68K,
	"tms320c64x": ArchitectureTMS320C64X,
	"m680x":      ArchitectureM680X,
	"evm":        ArchitectureEVM,
	"mos65xx":    ArchitectureMOS65XX,
	"6502":       ArchitectureMOS65XX,
	"wasm":       ArchitectureWASM,
	"bpf":        ArchitectureBPF,
	"ebpf":       ArchitectureBPF,
	"riscv":      ArchitectureRISCV,
	"risc-v":     ArchitectureRISCV,
	"sh":         ArchitectureSH,
	"tricore":    ArchitectureTriCore,
	"alpha":      ArchitectureAlpha,
}

var modeNames = map[string]Mode{
	"little":   ModeLittleEndian,
	"le":       ModeLittleEndian,
	"arm":      ModeARM,
	"16":       Mode16,
	"32":       Mode32,
	"64":       Mode64,
	"thumb":    ModeThumb,
	"mclass":   ModeMClass,
	"v8":       ModeV8,
	"micro":    ModeMicro,
	"mips3":    ModeMIPS3,
	"mips32r6": ModeMIPS32R6,
	"mips2":    ModeMIPS2,
	"v9":       ModeV9,
	"big":      ModeBigEndian,
	"be":       ModeBigEndian,
	"mips32":   ModeMIPS32,
	"mips64":   ModeMIPS64,
	"bpf":      ModeBPFClassic,
	"ebpf":     ModeBPFExtended,
	"riscv32":  ModeRISCV32,
	"riscv64":  ModeRISCV64,
	"riscvc":   ModeRISCVC,
}

var syntaxNames = map[string]Syntax{
	"default": SyntaxDefault,
	"intel":   SyntaxIntel,
	"att":     SyntaxATT,
	"at&t":    SyntaxATT,
	"noreg":   SyntaxNoReg,
	"masm":    SyntaxMASM,
}

// ParseArchitecture maps a name such as x86 or aarch64.
func ParseArchitecture(name string) (Architecture, error) {
	architecture, ok := architectureNames[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return 0, fmt.Errorf("unknown architecture %q", name)
	}
	return architecture, nil
}

// ParseMode maps a comma-separated list of mode bits such as 64 or thumb,v8.
func ParseMode(name string) (Mode, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, nil
	}
	var mode Mode
	for part := range strings.SplitSeq(name, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		bit, ok := modeNames[part]
		if !ok {
			return 0, fmt.Errorf("unknown mode %q", part)
		}
		mode |= bit
	}
	return mode, nil
}

// ParseSyntax maps a name such as intel or att.
func ParseSyntax(name string) (Syntax, error) {
	syntax, ok := syntaxNames[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return 0, fmt.Errorf("unknown syntax %q", name)
	}
	return syntax, nil
}

// Parse makes Architecture a command-line enum argument.
func (architecture *Architecture) Parse(name string) error {
	parsed, err := ParseArchitecture(name)
	if err != nil {
		return err
	}
	*architecture = parsed
	return nil
}

// Parse makes Mode a command-line argument (comma-separated bits).
func (mode *Mode) Parse(name string) error {
	parsed, err := ParseMode(name)
	if err != nil {
		return err
	}
	*mode = parsed
	return nil
}

// Parse makes Syntax a command-line enum argument.
func (syntax *Syntax) Parse(name string) error {
	parsed, err := ParseSyntax(name)
	if err != nil {
		return err
	}
	*syntax = parsed
	return nil
}
