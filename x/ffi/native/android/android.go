//go:build android

package android

import (
	"fmt"
	"sync"
	"syscall"

	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	noteMu      sync.Mutex
	notedVM     uintptr
	notedLoader int
)

// NoteVM records the JavaVM from JNI_OnLoad. Later calls skip dlopen.
func NoteVM(vm uintptr) {
	if vm == 0 {
		return
	}
	noteMu.Lock()
	notedVM = vm
	noteMu.Unlock()
}

// NoteLoader records the thread that loaded the library. That thread runs
// the main looper. Call it from JNI_OnLoad.
func NoteLoader() {
	tid := currentTID()
	if tid == 0 {
		return
	}
	noteMu.Lock()
	notedLoader = tid
	noteMu.Unlock()
}

func resetNote() {
	noteMu.Lock()
	notedVM = 0
	notedLoader = 0
	noteMu.Unlock()
}

func currentTID() int {
	id, _, _ := syscall.RawSyscall(syscall.SYS_GETTID, 0, 0, 0)
	return int(id)
}

// JavaVMs returns how many Java VMs this process has created.
// A VM noted at load time counts as one. The dlopen probe runs only when
// nothing noted a VM, because opening libnativehelper from another thread
// deadlocks the Android linker.
func JavaVMs() (int, error) {
	noteMu.Lock()
	noted := notedVM
	noteMu.Unlock()
	if noted != 0 {
		return 1, nil
	}
	got, err := native.Bind[JNI](native.Now, "libnativehelper.so", "libart.so")
	if err != nil {
		return 0, err
	}
	var vm uintptr
	var n int32
	if rc := got.JNIGetCreatedJavaVMs(&vm, 1, &n); rc != 0 {
		return 0, fmt.Errorf("JNI_GetCreatedJavaVMs: %d", rc)
	}
	return int(n), nil
}

// OnLooper reports whether this thread runs the Android main looper.
// The loader thread noted at JNI_OnLoad is that looper. libandroid is
// opened only when no loader thread was noted.
func OnLooper() (bool, error) {
	noteMu.Lock()
	tid := notedLoader
	noteMu.Unlock()
	if tid != 0 {
		return tid == currentTID(), nil
	}
	got, err := native.Bind[Looper](native.Now, "libandroid.so")
	if err != nil {
		return false, err
	}
	return got.ALooperForThread() != 0, nil
}
