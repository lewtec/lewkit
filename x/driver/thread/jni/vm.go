//go:build android && cgo

package jni

/*
#cgo LDFLAGS: -landroid
#include <dlfcn.h>
#include <jni.h>
#include <android/looper.h>

static int lewkit_java_vms(void) {
	void *handle = dlopen("libnativehelper.so", RTLD_NOW);
	if (handle == NULL) {
		handle = dlopen("libart.so", RTLD_NOW);
	}
	if (handle == NULL) {
		return 0;
	}
	typedef jint (*vms_fn)(JavaVM **, jsize, jsize *);
	vms_fn fn = (vms_fn)dlsym(handle, "JNI_GetCreatedJavaVMs");
	if (fn == NULL) {
		return 0;
	}
	JavaVM *vm = NULL;
	jsize n = 0;
	if (fn(&vm, 1, &n) != JNI_OK) {
		return 0;
	}
	return (int)n;
}

static int lewkit_on_looper(void) {
	return ALooper_forThread() != NULL;
}
*/
import "C"
import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	uithread "github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/thread/std"
)

func init() { driver.Register[uithread.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "thread_jni" }
func (factory) Name() string { return "Android looper" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if C.lewkit_java_vms() < 1 || C.lewkit_on_looper() == 0 {
		return fmt.Errorf("%w: no Java main looper", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (uithread.Driver, error) {
	// The looper is already pumping. Jobs still run on this thread when On
	// reports it; callers off the looper use the process queue until a later
	// post lands. The std queue is the fallback inside this process.
	return std.New(), nil
}
