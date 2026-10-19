//go:build ios

package std

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/native"
)

// beforeWaiting is kCFRunLoopBeforeWaiting. thread.Do runs before the loop sleeps.
const beforeWaiting = 1 << 5

var (
	iosBox atomic.Pointer[backend]
	iosCtx atomic.Value // context.Context

	loopOnce sync.Once
	loopErr  error

	runApp     func(argc int32, argv unsafe.Pointer, principal, delegate uintptr) int32
	mainLoop   func() uintptr
	wakeMain   func(uintptr)
	stopMain   func(uintptr)
	makeObs    func(allocator, activities uintptr, repeats byte, order int64, callout uintptr, ctx unsafe.Pointer) uintptr
	addObs     func(loop, observer, mode uintptr)
	commonMode uintptr
	observer   uintptr
	delegate   uintptr
	argBufs    [][]byte
	argvPtrs   []unsafe.Pointer
	turning    atomic.Bool
)

// loopIOS is the process main thread. UIApplicationMain owns the run loop so
// the system retires the launch screen. thread.Do wakes that loop.
func (t *backend) loopIOS(ctx context.Context) {
	iosBox.Store(t)
	iosCtx.Store(ctx)
	if err := bindUIKit(); err != nil {
		slog.Error(err.Error())
		t.serveJobs(ctx)
		return
	}
	go func() {
		<-ctx.Done()
		wakeLoop()
	}()
	slog.Info("ios application")
	code := runApp(uiArgs())
	slog.Info("ios application returned", "code", code)
	t.serveJobs(ctx)
}

func bindUIKit() error {
	loopOnce.Do(func() {
		loopErr = loadUIKit()
	})
	return loopErr
}

func loadUIKit() error {
	cf, err := native.Open("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", native.Global|native.Lazy)
	if err != nil {
		return err
	}
	ui, err := native.Open("/System/Library/Frameworks/UIKit.framework/UIKit", native.Global|native.Lazy)
	if err != nil {
		return err
	}
	if err := bind(cf, "CFRunLoopGetMain", &mainLoop); err != nil {
		return err
	}
	if err := bind(cf, "CFRunLoopWakeUp", &wakeMain); err != nil {
		return err
	}
	if err := bind(cf, "CFRunLoopStop", &stopMain); err != nil {
		return err
	}
	if err := bind(cf, "CFRunLoopObserverCreate", &makeObs); err != nil {
		return err
	}
	if err := bind(cf, "CFRunLoopAddObserver", &addObs); err != nil {
		return err
	}
	mode, err := native.Symbol(cf, "kCFRunLoopCommonModes")
	if err != nil {
		return err
	}
	commonMode = *(*uintptr)(unsafe.Pointer(mode))
	if err := bind(ui, "UIApplicationMain", &runApp); err != nil {
		return err
	}
	class, err := objc.RegisterClass(
		"LewkitAppDelegate",
		objc.GetClass("NSObject"),
		nil,
		nil,
		[]objc.MethodDef{{
			Cmd: objc.RegisterName("application:didFinishLaunchingWithOptions:"),
			Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, _ objc.ID) bool {
				return true
			},
		}},
	)
	if err != nil {
		return err
	}
	name := objc.ID(objc.GetClass("NSString")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithUTF8String:"), "LewkitAppDelegate")
	if name == 0 || class == 0 {
		return fmt.Errorf("%w: ios application delegate", driver.ErrUnavailable)
	}
	delegate = uintptr(name)
	callout := purego.NewCallback(uiKitObserver)
	observer = makeObs(0, beforeWaiting, 1, 0, callout, nil)
	loop := mainLoop()
	if observer == 0 || loop == 0 || commonMode == 0 {
		return fmt.Errorf("%w: ios run loop", driver.ErrUnavailable)
	}
	addObs(loop, observer, commonMode)
	return nil
}

func bind(lib uintptr, name string, fn any) error {
	sym, err := native.Symbol(lib, name)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", driver.ErrUnavailable, name, err)
	}
	native.Register(fn, sym)
	return nil
}

func uiArgs() (int32, unsafe.Pointer, uintptr, uintptr) {
	args := os.Args
	if len(args) == 0 {
		args = []string{"lewkit"}
	}
	argBufs = make([][]byte, len(args))
	argvPtrs = make([]unsafe.Pointer, len(args)+1)
	for i, arg := range args {
		argBufs[i] = append([]byte(arg), 0)
		argvPtrs[i] = unsafe.Pointer(&argBufs[i][0])
	}
	return int32(len(args)), unsafe.Pointer(&argvPtrs[0]), 0, delegate
}

func uiKitObserver(uintptr, uintptr, unsafe.Pointer) {
	box := iosBox.Load()
	if box == nil || !turning.CompareAndSwap(false, true) {
		return
	}
	defer turning.Store(false)
	if ctx, _ := iosCtx.Load().(context.Context); ctx != nil && ctx.Err() != nil {
		if stopMain != nil {
			stopMain(mainLoop())
		}
		return
	}
	select {
	case fn := <-box.jobs:
		fn()
	default:
	}
	box.runIdle()
}

func wakeLoop() {
	if wakeMain == nil || mainLoop == nil {
		return
	}
	wakeMain(mainLoop())
}
