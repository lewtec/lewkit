// Package udf is one UDF volume.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes or a symlink target. [Take]
// reads the volume. [Make] writes it. An El Torito ISO 9660 image is
// x/takemake/iso. The contract is in DESIGN.md beside the parent directory.
package udf
