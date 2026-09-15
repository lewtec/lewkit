package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/disasm"
)

type disasmCmd struct {
	architecture cmd.StringArg      `long:"architecture" default:"x86" help:"architecture" ctx:""`
	mode         cmd.StringArg      `long:"mode" default:"64" help:"comma-separated mode bits" ctx:""`
	syntax       cmd.StringArg      `long:"syntax" default:"intel" help:"assembly syntax" ctx:""`
	address      cmd.IntArg[uint64] `long:"address" default:"0" help:"start address" ctx:""`
	count        cmd.IntArg[uint]   `long:"count" default:"0" help:"instruction limit, 0 is all" ctx:""`
	skipData     cmd.Flag           `long:"skip-data" help:"skip undecodable bytes" ctx:""`
	hex          *disasmHexCmd
	raw          *disasmRawCmd
	file         *disasmFileCmd
}

func (disasmCmd) Description() string {
	return "disassemble machine code"
}

func (command *disasmCmd) Run(ctx context.Context) error {
	text, err := cmd.Usage[disasmCmd]("lewkit disasm")
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(text)
	return err
}

type disasmHexCmd struct {
	data cmd.StringArg `help:"hex bytes"`
}

func (disasmHexCmd) Description() string {
	return "disassemble a hex string"
}

func (command *disasmHexCmd) Run(ctx context.Context) error {
	code, err := disasm.DecodeHex(command.data.Value())
	if err != nil {
		return err
	}
	return writeDisassembly(ctx, code, cmd.Get[uint64](ctx, "address"))
}

type disasmRawCmd struct {
	path cmd.StringArg `help:"raw bytes path, or - for stdin"`
}

func (disasmRawCmd) Description() string {
	return "disassemble a raw byte file"
}

func (command *disasmRawCmd) Run(ctx context.Context) error {
	code, err := readInput(command.path.Value())
	if err != nil {
		return err
	}
	return writeDisassembly(ctx, code, cmd.Get[uint64](ctx, "address"))
}

type disasmFileCmd struct {
	path    cmd.StringArg `help:"elf, pe, or macho path, or - for stdin"`
	section cmd.StringArg `long:"section" default:"" help:"section name"`
}

func (disasmFileCmd) Description() string {
	return "disassemble a text section from an object file"
}

func (command *disasmFileCmd) Run(ctx context.Context) error {
	raw, err := readInput(command.path.Value())
	if err != nil {
		return err
	}
	text, err := disasm.ReadText(bytesReader(raw), int64(len(raw)), command.section.Value())
	if err != nil {
		return err
	}
	address := cmd.Get[uint64](ctx, "address")
	if address == 0 {
		address = text.Address
	}
	return writeDisassemblyArchitecture(ctx, text.Architecture, text.Mode, text.Bytes, address)
}

func writeDisassembly(ctx context.Context, code []byte, address uint64) error {
	architecture, err := disasm.ParseArchitecture(cmd.Get[string](ctx, "architecture"))
	if err != nil {
		return err
	}
	mode, err := disasm.ParseMode(cmd.Get[string](ctx, "mode"))
	if err != nil {
		return err
	}
	return writeDisassemblyArchitecture(ctx, architecture, mode, code, address)
}

func writeDisassemblyArchitecture(ctx context.Context, architecture disasm.Architecture, mode disasm.Mode, code []byte, address uint64) error {
	syntax, err := disasm.ParseSyntax(cmd.Get[string](ctx, "syntax"))
	if err != nil {
		return err
	}
	engine, err := disasm.Open(ctx, architecture, mode, disasm.WithSyntax(syntax), disasm.WithSkipData(cmd.Get[bool](ctx, "skip-data")))
	if err != nil {
		return err
	}
	defer engine.Close(ctx)

	limit := cmd.Get[uint](ctx, "count")
	var n uint
	for instruction, err := range engine.Iter(ctx, code, address) {
		if err != nil {
			return err
		}
		if _, err := os.Stdout.WriteString(formatInstruction(instruction)); err != nil {
			return err
		}
		n++
		if limit > 0 && n >= limit {
			break
		}
	}
	return nil
}

func formatInstruction(instruction disasm.Instruction) string {
	line := fmt.Sprintf("0x%08x  %-16s %s", instruction.Address, hex.EncodeToString(instruction.Bytes), instruction.Mnemonic)
	if instruction.Operands != "" {
		line += " " + instruction.Operands
	}
	return line + "\n"
}

func readInput(path string) ([]byte, error) {
	if path == "" || path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

type bytesReader []byte

func (data bytesReader) ReadAt(p []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, fmt.Errorf("negative offset")
	}
	if offset >= int64(len(data)) {
		return 0, io.EOF
	}
	n := copy(p, data[offset:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
