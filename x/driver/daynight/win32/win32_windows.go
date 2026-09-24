//go:build windows

package win32

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/event"
)

const (
	hkeyCurrentUser        = 0x80000001
	keyNotify              = 0x0010
	keyQueryValue          = 0x0001
	regNotifyChangeLastSet = 0x00000004
	regDWORD               = 4
)

var (
	advapi32  = syscall.NewLazyDLL("advapi32.dll")
	kernel32  = syscall.NewLazyDLL("kernel32.dll")
	regOpen   = advapi32.NewProc("RegOpenKeyExW")
	regQuery  = advapi32.NewProc("RegQueryValueExW")
	regNotify = advapi32.NewProc("RegNotifyChangeKeyValue")
	regClose  = advapi32.NewProc("RegCloseKey")
	createEv  = kernel32.NewProc("CreateEventW")
	waitEv    = kernel32.NewProc("WaitForSingleObject")
	closeH    = kernel32.NewProc("CloseHandle")

	personalize = `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`
)

type source struct {
	key    syscall.Handle
	mu     sync.Mutex
	scheme daynight.Mode
	bus    *event.Bus[daynight.Mode]
}

func available(context.Context) error {
	key, err := openKey()
	if err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	regClose.Call(uintptr(key))
	return nil
}

var (
	openOnce sync.Once
	openSrc  *source
	openErr  error
)

func open(ctx context.Context) (daynight.Driver, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	openOnce.Do(func() {
		key, err := openKey()
		if err != nil {
			openErr = err
			return
		}
		openSrc = &source{key: key, scheme: readTheme(key), bus: event.New[daynight.Mode]()}
		go openSrc.loop()
	})
	if openErr != nil {
		return nil, openErr
	}
	return openSrc, nil
}

func (src *source) Current(context.Context) (daynight.Mode, error) {
	src.mu.Lock()
	defer src.mu.Unlock()
	src.scheme = readTheme(src.key)
	return src.scheme, nil
}

func (src *source) Watch(ctx context.Context) (<-chan daynight.Mode, error) {
	src.mu.Lock()
	scheme := readTheme(src.key)
	src.scheme = scheme
	src.mu.Unlock()
	return daynight.Changes(ctx, scheme, src.bus.Subscribe(ctx)), nil
}

func (src *source) loop() {
	event, _, _ := createEv.Call(0, 0, 0, 0)
	if event == 0 {
		return
	}
	defer closeH.Call(event)
	for {
		status, _, _ := regNotify.Call(uintptr(src.key), 0, regNotifyChangeLastSet, event, 1)
		if status != 0 {
			return
		}
		waitEv.Call(event, 0xFFFFFFFF)
		scheme := readTheme(src.key)
		src.mu.Lock()
		same := src.scheme == scheme
		src.scheme = scheme
		src.mu.Unlock()
		if !same {
			src.bus.Publish(scheme)
		}
	}
}

func openKey() (syscall.Handle, error) {
	name, err := syscall.UTF16PtrFromString(personalize)
	if err != nil {
		return 0, err
	}
	var key syscall.Handle
	status, _, _ := regOpen.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(name)), 0, keyQueryValue|keyNotify, uintptr(unsafe.Pointer(&key)))
	if status != 0 {
		return 0, fmt.Errorf("registry: %w", syscall.Errno(status))
	}
	return key, nil
}

func readTheme(key syscall.Handle) daynight.Mode {
	name, err := syscall.UTF16PtrFromString("AppsUseLightTheme")
	if err != nil {
		return daynight.Light
	}
	var kind uint32
	var data uint32
	size := uint32(4)
	status, _, _ := regQuery.Call(
		uintptr(key),
		uintptr(unsafe.Pointer(name)),
		0,
		uintptr(unsafe.Pointer(&kind)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&size)),
	)
	if status != 0 || kind != regDWORD {
		return daynight.Light
	}
	return fromLightTheme(data)
}
