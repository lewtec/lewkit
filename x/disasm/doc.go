// Package disasm decodes machine code with Capstone running in wazero.
//
// [Open] creates an [Engine] for one [Architecture] and [Mode].
// [Engine.Iter] yields instructions from a byte slice.
// File formats are [ReadText]: ELF, PE, and Mach-O text sections.
// Hex input is [DecodeHex].
package disasm
