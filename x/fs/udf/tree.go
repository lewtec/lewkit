package udf

import (
	"io/fs"
	"time"

	"github.com/Xmister/udf"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

type dnode struct {
	*lewfs.PathNode[udf.File]
}

func newDir(name string) *dnode {
	return &dnode{PathNode: lewfs.NewPathDir[udf.File](name)}
}

func (n *dnode) add(rel string, uf *udf.File, dir bool) error {
	return n.Add(rel, uf, dir)
}

func (n *dnode) info() fs.FileInfo {
	return infoOf(n.PathNode)
}

func infoOf(n *lewfs.PathNode[udf.File]) fs.FileInfo {
	mode := fs.FileMode(0o444)
	var size int64
	var mod time.Time
	if n.IsDir() {
		mode = fs.ModeDir | 0o555
	}
	if uf := n.Payload(); uf != nil {
		size = uf.Size()
		mod = uf.ModTime()
		if !n.IsDir() {
			mode = uf.Mode() &^ fs.ModeDir
			if mode == 0 {
				mode = 0o444
			}
		}
	}
	return n.Info(size, mode, mod)
}
