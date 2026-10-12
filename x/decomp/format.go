package decomp

import (
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/disasm"
)

// Summary describes the image, its sections, and how many functions were recovered.
func (artifact *Artifact) Summary() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s %s %s entry 0x%x\n", artifact.format, artifact.architecture, modeLabel(artifact.mode), artifact.entry)
	fmt.Fprintf(&builder, "sections %d\n", len(artifact.sections))
	for _, section := range artifact.sections {
		fmt.Fprintf(&builder, "  0x%x %s %d%s\n", section.Address, section.Name, section.Size, sectionFlags(section))
	}
	fmt.Fprintf(&builder, "functions %d\n", len(artifact.functions))
	return builder.String()
}

// Listing prints the blocks and instructions of the function at address.
func (artifact *Artifact) Listing(address uint64) (string, error) {
	function, ok := artifact.Function(address)
	if !ok {
		return "", fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	names := artifact.displayNames()
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s:\n", function.Name)
	if function.Signature != "" {
		fmt.Fprintf(&builder, "signature %s\n", function.Signature)
	}
	if function.Comment != "" {
		fmt.Fprintf(&builder, "; %s\n", function.Comment)
	}
	for _, block := range function.Blocks {
		fmt.Fprintf(&builder, "; block 0x%x %s", block.Address, block.Kind)
		if block.Target != 0 {
			fmt.Fprintf(&builder, " -> 0x%x", block.Target)
		}
		if block.Fall != 0 {
			fmt.Fprintf(&builder, " fall 0x%x", block.Fall)
		}
		builder.WriteByte('\n')
		for _, instruction := range block.Instructions {
			builder.WriteString(disasm.FormatInstruction(instruction, names))
			if text, ok := artifact.Note(instruction.Address); ok {
				fmt.Fprintf(&builder, "; note %s\n", text)
			}
		}
	}
	for _, note := range artifact.notesIn(function.Address, function.Size) {
		if artifact.instructionAt(function, note.Address) {
			continue
		}
		fmt.Fprintf(&builder, "; note 0x%x %s\n", note.Address, note.Text)
	}
	for _, text := range function.Strings {
		fmt.Fprintf(&builder, "; string %q\n", text)
	}
	return builder.String(), nil
}

func (artifact *Artifact) notesIn(start, size uint64) []Note {
	var notes []Note
	for _, note := range artifact.Notes() {
		if size == 0 {
			if note.Address == start {
				notes = append(notes, note)
			}
			continue
		}
		if note.Address >= start && note.Address < start+size {
			notes = append(notes, note)
		}
	}
	return notes
}

func (artifact *Artifact) instructionAt(function Function, address uint64) bool {
	for _, block := range function.Blocks {
		for _, instruction := range block.Instructions {
			if instruction.Address == address {
				return true
			}
		}
	}
	return false
}

func (artifact *Artifact) displayNames() map[uint64]string {
	names := make(map[uint64]string, len(artifact.symbols)+len(artifact.functions))
	for address, name := range artifact.symbols {
		names[address] = name
	}
	for _, function := range artifact.functions {
		names[function.Address] = function.Name
	}
	return names
}

func (artifact *Artifact) nameAt(address uint64) string {
	if function := artifact.byAddress[address]; function != nil {
		return function.Name
	}
	if name := artifact.symbols[address]; name != "" {
		return name
	}
	return fmt.Sprintf("sub_%x", address)
}

func modeLabel(mode disasm.Mode) string {
	switch {
	case mode&disasm.Mode64 != 0:
		return "64"
	case mode&disasm.Mode32 != 0:
		return "32"
	case mode&disasm.Mode16 != 0:
		return "16"
	default:
		return fmt.Sprintf("%#x", uint32(mode))
	}
}

func sectionFlags(section disasm.Section) string {
	var flags []string
	if section.Read {
		flags = append(flags, "r")
	}
	if section.Write {
		flags = append(flags, "w")
	}
	if section.Exec {
		flags = append(flags, "x")
	}
	if len(flags) == 0 {
		return ""
	}
	return " " + strings.Join(flags, "")
}
