//go:build linux || windows

package wim

import (
	"io/fs"
	"time"

	winwim "github.com/Microsoft/go-winio/wim"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

type dnode struct {
	*lewfs.PathNode[winwim.File]
}

func newDir(name string) *dnode {
	return &dnode{PathNode: lewfs.NewPathDir[winwim.File](name)}
}

func (n *dnode) add(rel string, wf *winwim.File, dir bool) error {
	return n.Add(rel, wf, dir)
}

func (n *dnode) info() fs.FileInfo {
	return infoOf(n.PathNode)
}

func infoOf(n *lewfs.PathNode[winwim.File]) fs.FileInfo {
	mode := fs.FileMode(0o444)
	var size int64
	var mod time.Time
	if n.IsDir() {
		mode = fs.ModeDir | 0o555
	}
	if wf := n.Payload(); wf != nil {
		size = wf.Size
		mod = wf.LastWriteTime.Time()
	}
	return n.Info(size, mode, mod)
}
