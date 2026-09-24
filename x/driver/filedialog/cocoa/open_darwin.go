//go:build darwin

package cocoa

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/thread"
)

const (
	modalOK           = 1
	activationRegular = 0
	cocoaFramework    = "/System/Library/Frameworks/Cocoa.framework/Cocoa"
)

var (
	errPanel    = errors.New("file dialog panel")
	errPath     = errors.New("file dialog path")
	errNotBound = errors.New("file dialog: thread not bound")
	errNotMain  = errors.New("file dialog: not main thread")

	appOnce sync.Once
	appErr  error
)

var (
	selShared     = objc.RegisterName("sharedApplication")
	selSetPolicy  = objc.RegisterName("setActivationPolicy:")
	selActivate   = objc.RegisterName("activateIgnoringOtherApps:")
	selOpenPanel  = objc.RegisterName("openPanel")
	selSavePanel  = objc.RegisterName("savePanel")
	selFiles      = objc.RegisterName("setCanChooseFiles:")
	selDirs       = objc.RegisterName("setCanChooseDirectories:")
	selMultiple   = objc.RegisterName("setAllowsMultipleSelection:")
	selCreate     = objc.RegisterName("setCanCreateDirectories:")
	selTitle      = objc.RegisterName("setTitle:")
	selName       = objc.RegisterName("setNameFieldStringValue:")
	selTypes      = objc.RegisterName("setAllowedFileTypes:")
	selSetDir     = objc.RegisterName("setDirectoryURL:")
	selRun        = objc.RegisterName("runModal")
	selURLs       = objc.RegisterName("URLs")
	selURL        = objc.RegisterName("URL")
	selCount      = objc.RegisterName("count")
	selAt         = objc.RegisterName("objectAtIndex:")
	selPath       = objc.RegisterName("path")
	selFileRep    = objc.RegisterName("fileSystemRepresentation")
	selUTF8       = objc.RegisterName("UTF8String")
	selNew        = objc.RegisterName("new")
	selDrain      = objc.RegisterName("drain")
	selUTF8String = objc.RegisterName("stringWithUTF8String:")
	selArray      = objc.RegisterName("array")
	selAdd        = objc.RegisterName("addObject:")
	selFileURL    = objc.RegisterName("fileURLWithPath:isDirectory:")
)

func (opener) Choose(ctx context.Context, req filedialog.Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", filedialog.ErrRequest, err)
	}
	if !thread.Bound() {
		return nil, errNotBound
	}
	var (
		paths []string
		err   error
	)
	thread.Do(func() {
		if !thread.ProcessMain() {
			err = errNotMain
			return
		}
		paths, err = show(req)
	})
	return paths, err
}

func ensureApp() error {
	appOnce.Do(func() {
		if _, err := native.Open(cocoaFramework, native.Global|native.Lazy); err != nil {
			appErr = fmt.Errorf("%w: %w", errPanel, err)
			return
		}
		app := objc.ID(objc.GetClass("NSApplication")).Send(selShared)
		if app == 0 {
			appErr = errPanel
			return
		}
		app.Send(selSetPolicy, activationRegular)
		app.Send(selActivate, true)
	})
	return appErr
}

func show(req filedialog.Request) ([]string, error) {
	if err := ensureApp(); err != nil {
		return nil, err
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)

	panel := openPanel(req)
	if panel == 0 {
		return nil, errPanel
	}
	panel.Send(selTitle, nsString(req.TitleOrDefault()))
	panel.Send(selCreate, true)
	if req.Directory != "" {
		dir := objc.ID(objc.GetClass("NSURL")).Send(selFileURL, nsString(req.Directory), true)
		panel.Send(selSetDir, dir)
	}
	if types := fileTypes(req); types != 0 {
		panel.Send(selTypes, types)
	}
	if req.Save {
		if req.Name != "" {
			panel.Send(selName, nsString(req.Name))
		}
	} else {
		panel.Send(selFiles, !req.Folder)
		panel.Send(selDirs, req.Folder)
		panel.Send(selMultiple, req.Multiple)
	}
	if int(panel.Send(selRun)) != modalOK {
		return nil, filedialog.ErrCanceled
	}
	if req.Save {
		path := urlPath(panel.Send(selURL))
		if path == "" {
			return nil, errPath
		}
		return []string{path}, nil
	}
	urls := panel.Send(selURLs)
	count := int(urls.Send(selCount))
	paths := make([]string, 0, count)
	for i := range count {
		path := urlPath(urls.Send(selAt, i))
		if path == "" {
			return nil, errPath
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, errPath
	}
	return paths, nil
}

func openPanel(req filedialog.Request) objc.ID {
	if req.Save {
		return objc.ID(objc.GetClass("NSSavePanel")).Send(selSavePanel)
	}
	return objc.ID(objc.GetClass("NSOpenPanel")).Send(selOpenPanel)
}

func fileTypes(req filedialog.Request) objc.ID {
	names := filedialog.Extensions(req.Filters)
	if len(names) == 0 {
		return 0
	}
	list := objc.ID(objc.GetClass("NSMutableArray")).Send(selArray)
	for _, name := range names {
		list.Send(selAdd, nsString(name))
	}
	return list
}

func urlPath(url objc.ID) string {
	if url == 0 {
		return ""
	}
	if path := goString(url.Send(selFileRep)); path != "" {
		return path
	}
	return goString(url.Send(selPath).Send(selUTF8))
}

func nsString(text string) objc.ID {
	raw := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selUTF8String, unsafe.Pointer(&raw[0]))
}

func goString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	ptr := uintptr(id)
	if ptr == 0 {
		return ""
	}
	var buf []byte
	for {
		c := *(*byte)(unsafe.Pointer(ptr))
		if c == 0 {
			break
		}
		buf = append(buf, c)
		ptr++
	}
	return string(buf)
}
