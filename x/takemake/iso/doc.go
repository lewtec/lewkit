// Package iso is an ISO 9660 image.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes. A zero catalog is a data image.
// [File.Platform], [File.Emulation], or [File.Boot] opts into El Torito:
// the boot catalog and a FAT16 boot image. [Take] reads the image.
// [Make] writes it. A UDF volume is x/takemake/udf.
// The contract is in DESIGN.md beside the parent directory.
package iso
