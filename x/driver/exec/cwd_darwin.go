//go:build darwin

package exec

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	procPIDVnodePathInfo = 9
	// vnode_info is vinfo_stat (136) plus type, pad, and fsid.
	// vip_path follows it in vnode_info_path. pvi_cdir is first.
	vnodePathOffset       = 152
	maxPathLen            = 1024
	procVnodePathInfoSize = (vnodePathOffset + maxPathLen) * 2
)

var (
	procOnce sync.Once
	procInfo func(pid int32, flavor int32, arg uint64, buffer unsafe.Pointer, bufsize int32) int32
)

func cwdOf(pid int) (string, bool) {
	call := loadProcInfo()
	if call == nil {
		return "", false
	}
	buf := make([]byte, procVnodePathInfoSize)
	n := call(int32(pid), procPIDVnodePathInfo, 0, unsafe.Pointer(&buf[0]), int32(len(buf)))
	if n < vnodePathOffset+1 {
		return "", false
	}
	path := cString(buf[vnodePathOffset : vnodePathOffset+maxPathLen])
	if path == "" {
		return "", false
	}
	return path, true
}

func loadProcInfo() func(int32, int32, uint64, unsafe.Pointer, int32) int32 {
	procOnce.Do(func() {
		lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			return
		}
		sym, err := purego.Dlsym(lib, "proc_pidinfo")
		if err != nil {
			return
		}
		purego.RegisterFunc(&procInfo, sym)
	})
	return procInfo
}

func cString(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return ""
}
