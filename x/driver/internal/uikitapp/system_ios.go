//go:build ios

package uikitapp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver"
)

var (
	noteSeq atomic.Uint32
	opener  objc.ID

	selGeneral      = objc.RegisterName("generalPasteboard")
	selSetString    = objc.RegisterName("setString:")
	selData         = objc.RegisterName("dataWithBytes:length:")
	selImageData    = objc.RegisterName("imageWithData:")
	selSetImage     = objc.RegisterName("setImage:")
	selCurrent      = objc.RegisterName("currentDevice")
	selSetMonitor   = objc.RegisterName("setBatteryMonitoringEnabled:")
	selState        = objc.RegisterName("batteryState")
	selLevel        = objc.RegisterName("batteryLevel")
	selTrait        = objc.RegisterName("traitCollection")
	selStyle        = objc.RegisterName("userInterfaceStyle")
	selCurrentTrait = objc.RegisterName("currentTraitCollection")
	selMainScreen   = objc.RegisterName("mainScreen")
	selOpenURL      = objc.RegisterName("openURL:options:completionHandler:")
	selInteraction  = objc.RegisterName("interactionControllerWithURL:")
	selCenter       = objc.RegisterName("currentNotificationCenter")
	selAuth         = objc.RegisterName("requestAuthorizationWithOptions:completionHandler:")
	selSetTitle     = objc.RegisterName("setTitle:")
	selSetBody      = objc.RegisterName("setBody:")
	selSetSound     = objc.RegisterName("setSound:")
	selSetLevel     = objc.RegisterName("setInterruptionLevel:")
	selDefaultSound = objc.RegisterName("defaultSound")
	selRequest      = objc.RegisterName("requestWithIdentifier:content:trigger:")
	selAddNote      = objc.RegisterName("addNotificationRequest:withCompletionHandler:")
	selOpenMenu     = objc.RegisterName("presentOpenInMenuFromRect:inView:animated:")
)

// PasteText writes the pasteboard.
func PasteText(ctx context.Context, text string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var reply string
	err := onUI(func() {
		withPool(func() {
			reply = writeText(text)
		})
	})
	if err != nil {
		return "", err
	}
	return reply, nil
}

func writeText(text string) string {
	if err := ensure(); err != nil {
		return failText("pasteboard")
	}
	board := cls("UIPasteboard").Send(selGeneral)
	if board == 0 {
		return failText("pasteboard")
	}
	board.Send(selSetString, ns(text))
	return okText()
}

// PasteImage writes a PNG to the pasteboard.
func PasteImage(ctx context.Context, png []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(png) == 0 {
		return failText("bad image"), nil
	}
	var reply string
	err := onUI(func() {
		withPool(func() {
			reply = writeImage(png)
		})
	})
	if err != nil {
		return "", err
	}
	return reply, nil
}

func writeImage(png []byte) string {
	if err := ensure(); err != nil {
		return failText("pasteboard")
	}
	data := cls("NSData").Send(selData, unsafe.Pointer(&png[0]), uintptr(len(png)))
	image := cls("UIImage").Send(selImageData, data)
	if image == 0 {
		return failText("bad image")
	}
	board := cls("UIPasteboard").Send(selGeneral)
	if board == 0 {
		return failText("pasteboard")
	}
	board.Send(selSetImage, image)
	return okText()
}

// Battery reads UIDevice.
func Battery(ctx context.Context) (string, int, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	var status string
	var level int
	err := onUI(func() {
		if err := ensure(); err != nil {
			status = ""
			return
		}
		dev := cls("UIDevice").Send(selCurrent)
		if dev == 0 || levelFn == nil {
			return
		}
		dev.Send(selSetMonitor, true)
		status, level = batteryRead(int(dev.Send(selState)), levelFn(dev, selLevel))
	})
	if err != nil {
		return "", 0, err
	}
	if status == "" {
		return "", 0, fmt.Errorf("%w: battery", driver.ErrUnavailable)
	}
	return status, level, nil
}

// Appearance reads the interface style.
func Appearance(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var name string
	err := onUI(func() {
		if err := ensure(); err != nil {
			return
		}
		name = appearanceName(interfaceStyle())
	})
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("%w: appearance", driver.ErrUnavailable)
	}
	return name, nil
}

func interfaceStyle() int {
	if win := keyWindow(cls("UIApplication").Send(selShared)); win != 0 {
		if trait := win.Send(selTrait); trait != 0 {
			return int(trait.Send(selStyle))
		}
	}
	if screen := cls("UIScreen").Send(selMainScreen); screen != 0 {
		if trait := screen.Send(selTrait); trait != 0 {
			return int(screenStyle(trait))
		}
	}
	if trait := cls("UITraitCollection").Send(selCurrentTrait); trait != 0 {
		return int(trait.Send(selStyle))
	}
	return 0
}

func screenStyle(trait objc.ID) objc.ID { return trait.Send(selStyle) }

// Open opens a URL or file.
func Open(ctx context.Context, target string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return wait(ctx, func(done func(string)) {
		openTarget(target, hook(done))
	})
}

func openTarget(target string, finish func(string)) {
	withPool(func() {
		url := urlFrom(target)
		if url == 0 {
			finish(failText("bad target"))
			return
		}
		app := cls("UIApplication").Send(selShared)
		if app == 0 {
			finish(failText("could not open"))
			return
		}
		_ = url.Send(selRetain)
		block := objc.NewBlock(func(_ objc.Block, ok bool) {
			if ok {
				url.Send(selRelease)
				finish(okText())
				return
			}
			_ = onUI(func() {
				defer url.Send(selRelease)
				if presentFile(url) {
					finish(okText())
					return
				}
				finish(failText("could not open"))
			})
		})
		app.Send(selOpenURL, url, objc.ID(0), block)
		block.Release()
	})
}

func urlFrom(target string) objc.ID {
	text := strings.TrimSpace(target)
	if text == "" {
		return 0
	}
	if strings.HasPrefix(text, "/") {
		return cls("NSURL").Send(selFileURL, ns(text))
	}
	url := cls("NSURL").Send(selURLString, ns(text))
	if url == 0 || url.Send(selScheme) == 0 {
		return 0
	}
	return url
}

func presentFile(url objc.ID) bool {
	if url == 0 || url.Send(selIsFile) == 0 || menuFn == nil {
		return false
	}
	path := objcText(url.Send(selPath))
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	presenter := topController()
	view := presenter.Send(selView)
	if view == 0 {
		return false
	}
	doc := cls("UIDocumentInteractionController").Send(selInteraction, url)
	if doc == 0 {
		return false
	}
	_ = doc.Send(selRetain)
	if opener != 0 {
		opener.Send(selRelease)
	}
	opener = doc
	rect := viewBounds(view)
	mid := cgRect{
		Origin: cgPoint{X: rect.Size.Width / 2, Y: rect.Size.Height / 2},
		Size:   cgSize{Width: 1, Height: 1},
	}
	return menuFn(doc, selOpenMenu, mid, view, true)
}

// Notify posts a local notification.
func Notify(ctx context.Context, title, message, urgency string, id uint32) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return wait(ctx, func(done func(string)) {
		postNote(title, message, urgency, id, hook(done))
	})
}

func postNote(title, message, urgency string, id uint32, finish func(string)) {
	withPool(func() {
		center := cls("UNUserNotificationCenter").Send(selCenter)
		if center == 0 {
			finish(failText("notifications"))
			return
		}
		block := objc.NewBlock(func(_ objc.Block, granted bool, err objc.ID) {
			if err != 0 {
				text := errorText(err)
				if text == "" {
					text = "notifications"
				}
				finish(failText(text))
				return
			}
			if !granted {
				finish(failText("notifications denied"))
				return
			}
			_ = onUI(func() {
				deliver(center, title, message, urgency, id, finish)
			})
		})
		center.Send(selAuth, uintptr(1|2|4), block)
		block.Release()
	})
}

func deliver(center objc.ID, title, message, urgency string, id uint32, finish func(string)) {
	withPool(func() {
		content := cls("UNMutableNotificationContent").Send(selAlloc).Send(selInit)
		if content == 0 {
			finish(failText("notifications"))
			return
		}
		label := strings.TrimSpace(title)
		if label == "" {
			label = "Notification"
		}
		content.Send(selSetTitle, ns(label))
		content.Send(selSetBody, ns(message))
		if urgency != "low" {
			if sound := cls("UNNotificationSound").Send(selDefaultSound); sound != 0 {
				content.Send(selSetSound, sound)
			}
		}
		if responds(content, selSetLevel) {
			level := uintptr(1)
			switch urgency {
			case "low":
				level = 0
			case "critical":
				level = 2
			}
			content.Send(selSetLevel, level)
		}
		req := cls("UNNotificationRequest").Send(selRequest, ns(noteIdent(id)), content, objc.ID(0))
		if req == 0 {
			finish(failText("notifications"))
			return
		}
		block := objc.NewBlock(func(_ objc.Block, err objc.ID) {
			if err != 0 {
				text := errorText(err)
				if text == "" {
					text = "notifications"
				}
				finish(failText(text))
				return
			}
			finish(okText())
		})
		center.Send(selAddNote, req, block)
		block.Release()
	})
}

func noteIdent(id uint32) string {
	if id == 0 {
		id = noteSeq.Add(1)
	}
	return fmt.Sprintf("lewkit-%d", id)
}
