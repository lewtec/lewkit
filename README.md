# LEWKIT

Repository of well planned primitives to be used in other projects.

Disassembly is `x/disasm`: Capstone hosted in wazero. `lewkit disasm hex`, `lewkit disasm raw`, and `lewkit disasm file` print instructions.

Rushed code is not sustainable long term and this is our tool of reuse.

Test helpers live in `x/test`. Database open and migrate stay in `x/db`. CLI is `x/cmd`. Directed graphs are `x/graph` (DOT, Mermaid). Task trees with resource pools are `x/taskgroup`. Host capability selection is `x/driver`; a resizable window is `x/driver/window` (`Frame` / `Draw` swap, `Animate` ticks a paint func, `Subscribe` yields Resize/Expose/Close). The RGB triangle is `x/image.TriangleTurn`. `lewkit experiments window triangle` opens a window and animates it. `lewkit doctor` lists drivers as interface => implementation. Event buses are `x/event`. A locked OS-thread job queue is `x/thread` (`thread.Run` from main). Cgo-free `dlopen` is `x/ffi`. Filesystem names are `x/path`. Name predicates are `x/path/pick`. Stream codecs are `x/compression` (gzip, brotli, lz4, zstd, xz, bzip2). Container listings are `x/fs.Files`; `x/fs.New` indexes a listing into `io/fs`. `x/fs.Copy` writes a listing into a dest. `x/fs.Walk` turns an `io/fs` into a listing. UDF, WIM, ZIP, tar, and SquashFS adapters are `x/fs/udf`, `x/fs/wim`, `x/fs/zip`, `x/fs/tar`, and `x/fs/squashfs`.
