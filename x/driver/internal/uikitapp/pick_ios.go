//go:build ios

package uikitapp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ebitengine/purego/objc"
)

var (
	pickOnce     sync.Once
	pickErr      error
	pickClass    objc.Class
	pickMu       sync.Mutex
	liveDelegate objc.ID

	selInitOpen    = objc.RegisterName("initForOpeningContentTypes:asCopy:")
	selInitExport  = objc.RegisterName("initForExportingURLs:asCopy:")
	selInitTypes   = objc.RegisterName("initWithDocumentTypes:inMode:")
	selInitOne     = objc.RegisterName("initWithURL:inMode:")
	selSetMulti    = objc.RegisterName("setAllowsMultipleSelection:")
	selSetDir      = objc.RegisterName("setDirectoryURL:")
	selSetDelegate = objc.RegisterName("setDelegate:")
	selDidPick     = objc.RegisterName("documentPicker:didPickDocumentsAtURLs:")
	selCancelled   = objc.RegisterName("documentPickerWasCancelled:")
	selTypeExt     = objc.RegisterName("typeWithFilenameExtension:")
	selTypeID      = objc.RegisterName("typeWithIdentifier:")
	selStart       = objc.RegisterName("startAccessingSecurityScopedResource")
	selStop        = objc.RegisterName("stopAccessingSecurityScopedResource")
)

type pickHold struct {
	save   string
	remove bool
	finish func(string)
}

var picks sync.Map

// Pick shows the document picker.
func Pick(ctx context.Context, req PickRequest) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !lockGate() {
		return busyText(), nil
	}
	defer unlockGate()
	reply, err := wait(ctx, func(done func(string)) {
		showPick(req, hook(done))
	})
	if err != nil {
		releaseLive()
	}
	return reply, err
}

func showPick(req PickRequest, finish func(string)) {
	withPool(func() {
		if err := registerPicker(); err != nil {
			finish(failText("document picker"))
			return
		}
		presenter := topController()
		if presenter == 0 {
			finish(idleText())
			return
		}
		picker, hold, err := makePicker(req)
		if err != nil || picker == 0 {
			if hold.remove && hold.save != "" {
				_ = os.Remove(hold.save)
			}
			finish(failText("document picker"))
			return
		}
		box := objc.ID(pickClass).Send(selAlloc).Send(selInit)
		if box == 0 {
			if hold.remove && hold.save != "" {
				_ = os.Remove(hold.save)
			}
			finish(failText("document picker"))
			return
		}
		hold.finish = finish
		picks.Store(uintptr(box), &hold)
		pickMu.Lock()
		liveDelegate = box
		pickMu.Unlock()
		picker.Send(selSetDelegate, box)
		if !req.Save {
			picker.Send(selSetMulti, req.Multiple && !req.Folder)
		}
		if req.Directory != "" {
			picker.Send(selSetDir, cls("NSURL").Send(selFileURL, ns(req.Directory)))
		}
		present(presenter, picker)
		picker.Send(selRelease)
	})
}

func registerPicker() error {
	pickOnce.Do(func() {
		class, err := objc.RegisterClass(
			"LewkitDocPicker",
			objc.GetClass("NSObject"),
			nil,
			nil,
			[]objc.MethodDef{
				{Cmd: selDidPick, Fn: pickDid},
				{Cmd: selCancelled, Fn: pickCancel},
			},
		)
		pickClass = class
		pickErr = err
	})
	return pickErr
}

func pickDid(self objc.ID, _ objc.SEL, _ objc.ID, urls objc.ID) {
	hold, ok := takePick(self)
	if !ok {
		return
	}
	if hold.save != "" {
		hold.finish(okBody(pathJSON([]string{hold.save})))
		return
	}
	var paths []string
	n := 0
	if urls != 0 {
		n = int(urls.Send(selCount))
	}
	if n > 32 {
		n = 32
	}
	for i := range n {
		url := urls.Send(selAt, uintptr(i))
		if path := keepURL(url); path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		hold.finish(cancelText())
		return
	}
	hold.finish(okBody(pathJSON(paths)))
}

func pickCancel(self objc.ID, _ objc.SEL, _ objc.ID) {
	hold, ok := takePick(self)
	if !ok {
		return
	}
	if hold.remove && hold.save != "" {
		_ = os.Remove(hold.save)
	}
	hold.finish(cancelText())
}

func takePick(self objc.ID) (*pickHold, bool) {
	pickMu.Lock()
	if liveDelegate == self {
		liveDelegate = 0
	}
	pickMu.Unlock()
	value, ok := picks.LoadAndDelete(uintptr(self))
	if !ok {
		return nil, false
	}
	hold, _ := value.(*pickHold)
	return hold, hold != nil
}

func releaseLive() {
	pickMu.Lock()
	self := liveDelegate
	liveDelegate = 0
	pickMu.Unlock()
	if self == 0 {
		return
	}
	_ = onUI(func() {
		if hold, ok := takePick(self); ok && hold.remove && hold.save != "" {
			_ = os.Remove(hold.save)
		}
	})
}

func makePicker(req PickRequest) (objc.ID, pickHold, error) {
	class := cls("UIDocumentPickerViewController")
	if req.Save {
		return exportPicker(class, req)
	}
	return openPicker(class, req)
}

func exportPicker(class objc.ID, req PickRequest) (objc.ID, pickHold, error) {
	dir, err := inbox()
	if err != nil {
		return 0, pickHold{}, err
	}
	path := filepath.Join(dir, safeName(req.Name))
	hold := pickHold{save: path}
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return 0, pickHold{}, err
		}
		hold.remove = true
	}
	url := cls("NSURL").Send(selFileURL, ns(path))
	if instances(class, selInitExport) {
		urls := cls("NSArray").Send(selArrayOne, url)
		picker := class.Send(selAlloc).Send(selInitExport, urls, true)
		if picker != 0 {
			return picker, hold, nil
		}
	}
	if instances(class, selInitOne) {
		picker := class.Send(selAlloc).Send(selInitOne, url, uintptr(2))
		if picker != 0 {
			return picker, hold, nil
		}
	}
	if hold.remove {
		_ = os.Remove(path)
	}
	return 0, pickHold{}, os.ErrNotExist
}

func openPicker(class objc.ID, req PickRequest) (objc.ID, pickHold, error) {
	if instances(class, selInitOpen) {
		types := contentTypes(req)
		if types != 0 {
			picker := class.Send(selAlloc).Send(selInitOpen, types, !req.Folder)
			if picker != 0 {
				return picker, pickHold{}, nil
			}
		}
	}
	names := legacyTypes(req)
	if len(names) == 0 || !instances(class, selInitTypes) {
		return 0, pickHold{}, os.ErrNotExist
	}
	list := cls("NSMutableArray").Send(selArray)
	for _, name := range names {
		list.Send(selAdd, ns(name))
	}
	mode := uintptr(0)
	if req.Folder {
		mode = 1
	}
	picker := class.Send(selAlloc).Send(selInitTypes, list, mode)
	return picker, pickHold{}, nil
}

func contentTypes(req PickRequest) objc.ID {
	if cls("UTType") == 0 {
		return 0
	}
	list := cls("NSMutableArray").Send(selArray)
	if req.Folder {
		if item := cls("UTType").Send(selTypeID, ns("public.folder")); item != 0 {
			list.Send(selAdd, item)
		}
		return list
	}
	for _, ext := range req.Exts {
		ext = trimExt(ext)
		if ext == "" {
			continue
		}
		if item := cls("UTType").Send(selTypeExt, ns(ext)); item != 0 {
			list.Send(selAdd, item)
		}
	}
	if int(list.Send(selCount)) == 0 {
		if item := cls("UTType").Send(selTypeID, ns("public.content")); item != 0 {
			list.Send(selAdd, item)
		}
	}
	if int(list.Send(selCount)) == 0 {
		return 0
	}
	return list
}

func legacyTypes(req PickRequest) []string {
	if req.Folder {
		return []string{"public.folder"}
	}
	return []string{"public.content"}
}

func trimExt(ext string) string {
	return strings.TrimPrefix(strings.TrimSpace(ext), ".")
}

func keepURL(url objc.ID) string {
	if url == 0 {
		return ""
	}
	scoped := url.Send(selStart) != 0
	if scoped {
		defer url.Send(selStop)
	}
	path := objcText(url.Send(selPath))
	if path == "" {
		return ""
	}
	kept, err := keepFile(path)
	if err != nil {
		if scoped {
			return path
		}
		return ""
	}
	return kept
}

func keepFile(src string) (string, error) {
	dir, err := inbox()
	if err != nil {
		return "", err
	}
	base := safeName(filepath.Base(src))
	dest := filepath.Join(dir, base)
	if _, err := os.Stat(dest); err == nil {
		dest = filepath.Join(dir, time.Now().UTC().Format("20060102150405.000000000")+"-"+base)
	}
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return "", closeErr
	}
	return dest, nil
}

func inbox() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "lewkit-inbox")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func pathJSON(paths []string) string {
	raw, err := json.Marshal(paths)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
