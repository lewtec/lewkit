// Package disasm decodes machine code with Capstone running in wazero.
//
// [Open] creates an [Engine] for one [Architecture] and [Mode].
// [Engine.Iter] yields instructions from a byte slice.
// File formats are [ReadText]: ELF, PE, and Mach-O text sections
// plus their symbol tables. [FormatInstruction] prints labels and
// operand symbol refs.
// Hex input is [DecodeHex].
package disasm
