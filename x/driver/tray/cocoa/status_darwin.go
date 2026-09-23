//go:build darwin

package cocoa

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/thread"
)

const (
	statusLength         = float64(-1) // NSVariableStatusItemLength
	activationAccessory  = 1
	activationProhibited = 2
	controlStateOn       = 1
	controlStateOff      = 0
)

var (
	appOnce sync.Once
	appErr  error

	selShared          = objc.RegisterName("sharedApplication")
	selPolicy          = objc.RegisterName("activationPolicy")
	selSetPolicy       = objc.RegisterName("setActivationPolicy:")
	selNextEvent       = objc.RegisterName("nextEventMatchingMask:untilDate:inMode:dequeue:")
	selSendEvent       = objc.RegisterName("sendEvent:")
	selDistantPast     = objc.RegisterName("distantPast")
	selNew             = objc.RegisterName("new")
	selDrain           = objc.RegisterName("drain")
	selRetain          = objc.RegisterName("retain")
	selRelease         = objc.RegisterName("release")
	selUTF8            = objc.RegisterName("stringWithUTF8String:")
	selSystemStatusBar = objc.RegisterName("systemStatusBar")
	selStatusItem      = objc.RegisterName("statusItemWithLength:")
	selRemoveItem      = objc.RegisterName("removeStatusItem:")
	selButton          = objc.RegisterName("button")
	selSetImage        = objc.RegisterName("setImage:")
	selSetToolTip      = objc.RegisterName("setToolTip:")
	selDataWithBytes   = objc.RegisterName("dataWithBytes:length:")
	selInitWithData    = objc.RegisterName("initWithData:")
	selSetSize         = objc.RegisterName("setSize:")
	selSetTemplate     = objc.RegisterName("setTemplate:")
	selAlloc           = objc.RegisterName("alloc")
	selInit            = objc.RegisterName("init")
	selSetAutoenables  = objc.RegisterName("setAutoenablesItems:")
	selAddItem         = objc.RegisterName("addItem:")
	selSetMenu         = objc.RegisterName("setMenu:")
	selSeparator       = objc.RegisterName("separatorItem")
	selInitItem        = objc.RegisterName("initWithTitle:action:keyEquivalent:")
	selSetTarget       = objc.RegisterName("setTarget:")
	selSetTag          = objc.RegisterName("setTag:")
	selSetEnabled      = objc.RegisterName("setEnabled:")
	selSetState        = objc.RegisterName("setState:")
	selSetSubmenu      = objc.RegisterName("setSubmenu:")
	selTag             = objc.RegisterName("tag")
	selActivate        = objc.RegisterName("trayActivate:")

	runLoopMode objc.ID
	targetClass objc.Class
	targets     sync.Map
)

type nsSize struct {
	Width, Height float64
}

var (
	errCocoa       = errors.New("cocoa")
	errStatusItem  = errors.New("status item")
	errTrayTarget  = errors.New("tray target")
	errStatusImage = errors.New("status image")
)

type opener struct{}

func (opener) Open(ctx context.Context, cfg tray.Config) (tray.Tray, error) {
	if err := startApp(); err != nil {
		return nil, err
	}
	item := &statusItem{}
	var openErr error
	thread.Do(func() {
		openErr = item.create(cfg)
	})
	if openErr != nil {
		return nil, openErr
	}
	context.AfterFunc(ctx, func() {
		if err := item.Close(); err != nil {
			return
		}
	})
	return item, nil
}

type statusItem struct {
	mu       sync.Mutex
	cfg      tray.Config
	item     objc.ID
	target   objc.ID
	closed   bool
	closeOne sync.Once
}

func startApp() error {
	if !thread.Bound() {
		return tray.ErrNotBound
	}
	var err error
	thread.Do(func() {
		if !thread.ProcessMain() {
			err = tray.ErrNotMain
			return
		}
		appOnce.Do(func() {
			if _, openErr := native.Open("/System/Library/Frameworks/Cocoa.framework/Cocoa", native.Global|native.Lazy); openErr != nil {
				appErr = fmt.Errorf("%w: %w", errCocoa, openErr)
				return
			}
			class, classErr := objc.RegisterClass(
				"LewkitTrayTarget",
				objc.GetClass("NSObject"),
				nil,
				nil,
				[]objc.MethodDef{{
					Cmd: selActivate,
					Fn:  trayActivate,
				}},
			)
			if classErr != nil {
				appErr = classErr
				return
			}
			targetClass = class
			runLoopMode = nsString("kCFRunLoopDefaultMode").Send(selRetain)
			app := objc.ID(objc.GetClass("NSApplication")).Send(selShared)
			if int(app.Send(selPolicy)) == activationProhibited {
				app.Send(selSetPolicy, activationAccessory)
			}
			thread.OnIdle(func() { pump(app) })
		})
		if err == nil {
			err = appErr
		}
	})
	return err
}

func trayActivate(self objc.ID, _ objc.SEL, sender objc.ID) {
	value, ok := targets.Load(uintptr(self))
	if !ok {
		return
	}
	value.(*statusItem).click(int(sender.Send(selTag)))
}

func pump(app objc.ID) {
	if !thread.ProcessMain() {
		return
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)
	date := objc.ID(objc.GetClass("NSDate")).Send(selDistantPast)
	for {
		event := app.Send(selNextEvent, ^uintptr(0), date, runLoopMode, true)
		if event == 0 {
			break
		}
		app.Send(selSendEvent, event)
	}
}

func (item *statusItem) create(cfg tray.Config) error {
	bar := objc.ID(objc.GetClass("NSStatusBar")).Send(selSystemStatusBar)
	status := bar.Send(selStatusItem, statusLength)
	if status == 0 {
		return errStatusItem
	}
	target := objc.ID(targetClass).Send(selAlloc).Send(selInit)
	if target == 0 {
		bar.Send(selRemoveItem, status)
		return errTrayTarget
	}
	item.item = status
	item.target = target
	targets.Store(uintptr(target), item)
	if err := item.apply(cfg); err != nil {
		bar.Send(selRemoveItem, status)
		targets.Delete(uintptr(target))
		target.Send(selRelease)
		return err
	}
	return nil
}

func (item *statusItem) Update(cfg tray.Config) error {
	if cfg.ID == "" {
		item.mu.Lock()
		cfg.ID = item.cfg.ID
		item.mu.Unlock()
	}
	var err error
	thread.Do(func() {
		item.mu.Lock()
		closed := item.closed
		item.mu.Unlock()
		if closed {
			err = tray.ErrClosed
			return
		}
		err = item.apply(cfg)
	})
	return err
}

func (item *statusItem) Close() error {
	item.closeOne.Do(func() {
		thread.Do(func() {
			item.mu.Lock()
			item.closed = true
			status := item.item
			target := item.target
			item.mu.Unlock()
			if status != 0 {
				objc.ID(objc.GetClass("NSStatusBar")).Send(selSystemStatusBar).Send(selRemoveItem, status)
			}
			if target != 0 {
				targets.Delete(uintptr(target))
				target.Send(selRelease)
			}
		})
	})
	return nil
}

func (item *statusItem) apply(cfg tray.Config) error {
	button := item.item.Send(selButton)
	if err := setButtonImage(button, cfg.Icon); err != nil {
		return err
	}
	button.Send(selSetToolTip, nsString(tray.Tip(cfg)))
	menu, err := buildMenu(tray.Tree(cfg.Menu), item.target)
	if err != nil {
		return err
	}
	item.item.Send(selSetMenu, menu)
	item.mu.Lock()
	item.cfg = cfg
	item.mu.Unlock()
	return nil
}

func (item *statusItem) click(id int) {
	item.mu.Lock()
	nodes := tray.Tree(item.cfg.Menu)
	item.mu.Unlock()
	clicked, ok := tray.Find(nodes, id)
	if !ok || clicked.OnClick == nil || clicked.Separator || len(clicked.Children) > 0 {
		return
	}
	go clicked.OnClick()
}

func setButtonImage(button objc.ID, icon tray.Icon) error {
	img, err := tray.Raster(icon, 36)
	if err != nil {
		return err
	}
	if img == nil {
		button.Send(selSetImage, objc.ID(0))
		return nil
	}
	raw, err := tray.EncodePNG(img)
	if err != nil {
		return err
	}
	data := objc.ID(objc.GetClass("NSData")).Send(selDataWithBytes, unsafe.Pointer(&raw[0]), len(raw))
	image := objc.ID(objc.GetClass("NSImage")).Send(selAlloc).Send(selInitWithData, data)
	if image == 0 {
		return fmt.Errorf("%w: %w", tray.ErrIcon, errStatusImage)
	}
	image.Send(selSetSize, nsSize{Width: 18, Height: 18})
	image.Send(selSetTemplate, false)
	button.Send(selSetImage, image)
	return nil
}

func buildMenu(nodes []tray.Node, target objc.ID) (objc.ID, error) {
	menu := objc.ID(objc.GetClass("NSMenu")).Send(selAlloc).Send(selInit)
	menu.Send(selSetAutoenables, false)
	for _, node := range nodes {
		if err := addItem(menu, node, target); err != nil {
			return 0, err
		}
	}
	return menu, nil
}

func addItem(menu objc.ID, node tray.Node, target objc.ID) error {
	if node.Item.Separator {
		menu.Send(selAddItem, objc.ID(objc.GetClass("NSMenuItem")).Send(selSeparator))
		return nil
	}
	row := objc.ID(objc.GetClass("NSMenuItem")).Send(selAlloc).Send(
		selInitItem,
		nsString(node.Item.Label),
		selActivate,
		nsString(""),
	)
	row.Send(selSetTarget, target)
	row.Send(selSetTag, node.ID)
	row.Send(selSetEnabled, !node.Item.Disabled)
	state := controlStateOff
	if node.Item.Checked {
		state = controlStateOn
	}
	row.Send(selSetState, state)
	if len(node.Children) > 0 {
		child, err := buildMenu(node.Children, target)
		if err != nil {
			return err
		}
		row.Send(selSetSubmenu, child)
	}
	menu.Send(selAddItem, row)
	return nil
}

func nsString(text string) objc.ID {
	raw := append([]byte(text), 0)
	return objc.ID(objc.GetClass("NSString")).Send(selUTF8, unsafe.Pointer(&raw[0]))
}
