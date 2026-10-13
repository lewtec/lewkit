//go:build darwin

package cocoa

import (
	"context"
	"errors"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/driver/internal/cocoaalert"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/prelude"
	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	errNoItems  = errors.New("no items")
	errNotBound = errors.New("alert: thread not bound")
)

var (
	selAlloc   = objc.RegisterName("alloc")
	selInit    = objc.RegisterName("init")
	selMessage = objc.RegisterName("setMessageText:")
	selButton  = objc.RegisterName("addButtonWithTitle:")
	selLayout  = objc.RegisterName("layout")
	selView    = objc.RegisterName("setAccessoryView:")
	selRun     = objc.RegisterName("runModal")
	selValue   = objc.RegisterName("stringValue")
	selNew     = objc.RegisterName("new")
	selDrain   = objc.RegisterName("drain")
	selFrame   = objc.RegisterName("initWithFrame:")
)

type nsPoint struct{ X, Y float64 }
type nsSize struct{ Width, Height float64 }
type nsRect struct {
	Origin nsPoint
	Size   nsSize
}

func (backend) Choose(ctx context.Context, opts launcher.ChooseOptions) (*launcher.Item, error) {
	var (
		item *launcher.Item
		err  error
	)
	err = onUI(ctx, func() error {
		item, err = choose(opts)
		return err
	})
	return item, err
}

func (backend) Prompt(ctx context.Context, prompt string) (string, error) {
	var text string
	err := onUI(ctx, func() error {
		var err error
		text, err = ask(prompt)
		return err
	})
	return text, err
}

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	var yes bool
	err := onUI(ctx, func() error {
		var err error
		yes, err = confirm(message)
		return err
	})
	return yes, err
}

func onUI(ctx context.Context, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !thread.Bound() {
		return errNotBound
	}
	var err error
	thread.Do(func() { err = fn() })
	return err
}

func choose(opts launcher.ChooseOptions) (*launcher.Item, error) {
	labels := make([]string, 0, len(opts.Items))
	for _, item := range opts.Items {
		if label := launcher.ItemLabel(item); label != "" {
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return nil, errNoItems
	}
	var picked *launcher.Item
	title := opts.Prompt
	if title == "" {
		title = "Choose"
	}
	err := withAlert(title, func(alert objc.ID) error {
		for _, label := range labels {
			alert.Send(selButton, native.NSString(label))
		}
		alert.Send(selButton, native.NSString("Cancel"))
		index, ok := buttonIndex(modal(alert), len(labels))
		if !ok {
			return nil
		}
		picked = launcher.MatchSelected(opts.Items, labels[index])
		return nil
	})
	return picked, err
}

func ask(prompt string) (string, error) {
	var text string
	title := prompt
	if title == "" {
		title = "Input"
	}
	err := withAlert(title, func(alert objc.ID) error {
		field := objc.ID(objc.GetClass("NSTextField")).Send(selAlloc).Send(selFrame, nsRect{Size: nsSize{Width: 240, Height: 24}})
		if field == 0 {
			return cocoaalert.ErrAlert
		}
		alert.Send(selView, field)
		alert.Send(selButton, native.NSString("OK"))
		alert.Send(selButton, native.NSString("Cancel"))
		alert.Send(selLayout)
		if _, ok := buttonIndex(modal(alert), 1); !ok {
			return launcher.ErrCanceled
		}
		text = native.UTF8String(field.Send(selValue))
		return nil
	})
	return text, err
}

func confirm(message string) (bool, error) {
	var yes bool
	title := message
	if title == "" {
		title = "Confirm"
	}
	err := withAlert(title, func(alert objc.ID) error {
		alert.Send(selButton, native.NSString("Yes"))
		alert.Send(selButton, native.NSString("No"))
		index, ok := buttonIndex(modal(alert), 1)
		yes = ok && index == 0
		return nil
	})
	return yes, err
}

func withAlert(title string, fn func(objc.ID) error) error {
	if err := cocoaalert.Open(); err != nil {
		return err
	}
	pool := objc.ID(objc.GetClass("NSAutoreleasePool")).Send(selNew)
	defer pool.Send(selDrain)
	alert := objc.ID(objc.GetClass("NSAlert")).Send(selAlloc).Send(selInit)
	if alert == 0 {
		return cocoaalert.ErrAlert
	}
	alert.Send(selMessage, native.NSString(title))
	return fn(alert)
}

func modal(alert objc.ID) int {
	return int(int64(alert.Send(selRun)))
}
