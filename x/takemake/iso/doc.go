// Package iso is one El Torito image.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes. The ISO 9660 directory, the
// El Torito catalog, and the FAT16 boot image are that file. [Take]
// reads the image. [Make] writes it. A UDF volume is x/takemake/udf.
// The contract is in DESIGN.md beside the parent directory. This
// scaffold returns [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package iso
