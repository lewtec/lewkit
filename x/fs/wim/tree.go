//go:build linux || windows

package wim

import (
	"io/fs"
	"time"

	winwim "github.com/Microsoft/go-winio/wim"

	"github.com/lewtec/lewkit/x/fs/internal/dtree"
)

type dnode = dtree.Node[*winwim.File]

func newDir(name string) *dnode {
	return dtree.NewDir[*winwim.File](name)
}

func nodeInfo(n *dnode) fs.FileInfo {
	mode := fs.FileMode(0o444)
	var size int64
	var mod time.Time
	if n.Dir {
		mode = fs.ModeDir | 0o555
	}
	if n.Val != nil {
		size = n.Val.Size
		mod = n.Val.LastWriteTime.Time()
	}
	return dtree.Info(n.Name, size, mode, mod)
}
