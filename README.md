# LEWKIT

Repository of well planned primitives to be used in other projects.

Rushed code is not sustainable long term and this is our tool of reuse.

Test helpers live in `x/test`. Database open and migrate stay in `x/db`. CLI is `x/cmd`. Filesystem names are `x/path`. Stream codecs are `x/compression` (gzip, brotli, lz4, zstd, xz, bzip2). Container listings are `x/fs.Files`; `x/fs.New` indexes a listing into `io/fs`. `x/fs.Copy` writes one `io/fs` into another. `x/fs.CopyFiles` writes a listing in order. UDF, WIM, ZIP, tar, and SquashFS adapters are `x/fs/udf`, `x/fs/wim`, `x/fs/zip`, `x/fs/tar`, and `x/fs/squashfs`.
