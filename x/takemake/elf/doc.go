// Package elf is one ELF image.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: class, endian, machine, section, symbol, reloc, and segment.
// [Take] reads an ELF into that file. [Make] writes it. A composed file
// and a taken file are the same type. The contract is in DESIGN.md
// beside the parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrUnsupported].
package elf
