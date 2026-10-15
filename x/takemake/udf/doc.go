// Package udf is one UDF volume.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an entry: name, mode, time, and bytes. [Take] reads the volume.
// [Make] writes it. An El Torito ISO 9660 image is x/takemake/iso.
// The contract is in DESIGN.md beside the parent directory. This
// scaffold returns [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package udf
