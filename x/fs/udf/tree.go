package udf

import (
	"io/fs"
	"time"

	"github.com/Xmister/udf"

	"github.com/lewtec/lewkit/x/fs/internal/dtree"
)

type dnode = dtree.Node[*udf.File]

func newDir(name string) *dnode {
	return dtree.NewDir[*udf.File](name)
}

func nodeInfo(n *dnode) fs.FileInfo {
	mode := fs.FileMode(0o444)
	var size int64
	var mod time.Time
	if n.Dir {
		mode = fs.ModeDir | 0o555
	}
	if n.Val != nil {
		size = n.Val.Size()
		mod = n.Val.ModTime()
		if !n.Dir {
			mode = n.Val.Mode() &^ fs.ModeDir
			if mode == 0 {
				mode = 0o444
			}
		}
	}
	return dtree.Info(n.Name, size, mode, mod)
}
