//go:build darwin

package cocoa

import (
	"context"
	"errors"
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/chooser"
	"github.com/lewtec/lewkit/x/thread"
)

const modalOK = 1

var (
	errPanel    = errors.New("chooser panel")
	errNotBound = errors.New("chooser: thread not bound")
)

var (
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
	selUTF8       = objc.RegisterName("UTF8String")
	selNew        = objc.RegisterName("new")
	selDrain      = objc.RegisterName("drain")
	selUTF8String = objc.RegisterName("stringWithUTF8String:")
	selArray      = objc.RegisterName("array")
	selAdd        = objc.RegisterName("addObject:")
	selFileURL    = objc.RegisterName("fileURLWithPath:isDirectory:")
)

func (opener) Choose(ctx context.Context, req chooser.Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", chooser.ErrRequest, err)
	}
	if !thread.Bound() {
		return nil, errNotBound
	}
	var (
		paths []string
		err   error
	)
	thread.Do(func() {
		paths, err = show(req)
	})
	return paths, err
}

func show(req chooser.Request) ([]string, error) {
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
		return nil, chooser.ErrCanceled
	}
	if req.Save {
		path := goString(panel.Send(selURL).Send(selPath).Send(selUTF8))
		if path == "" {
			return nil, errPanel
		}
		return []string{path}, nil
	}
	urls := panel.Send(selURLs)
	count := int(urls.Send(selCount))
	paths := make([]string, 0, count)
	for i := range count {
		path := goString(urls.Send(selAt, i).Send(selPath).Send(selUTF8))
		if path == "" {
			return nil, errPanel
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, errPanel
	}
	return paths, nil
}

func openPanel(req chooser.Request) objc.ID {
	if req.Save {
		return objc.ID(objc.GetClass("NSSavePanel")).Send(selSavePanel)
	}
	return objc.ID(objc.GetClass("NSOpenPanel")).Send(selOpenPanel)
}

func fileTypes(req chooser.Request) objc.ID {
	names := chooser.Extensions(req.Filters)
	if len(names) == 0 {
		return 0
	}
	list := objc.ID(objc.GetClass("NSMutableArray")).Send(selArray)
	for _, name := range names {
		list.Send(selAdd, nsString(name))
	}
	return list
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
