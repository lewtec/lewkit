package host

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/hostask"
	"github.com/lewtec/lewkit/x/driver/internal/uikitapp"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

type uikitFactory struct{}

func (uikitFactory) ID() string   { return "messagebox_uikit" }
func (uikitFactory) Name() string { return "UIKit" }
func (uikitFactory) Weight() int  { return 80 }

func (uikitFactory) CheckCompatibility(context.Context) error { return uikitapp.RequireProcess() }

func (uikitFactory) New(context.Context) (messagebox.Driver, error) { return uikitBox{}, nil }

type uikitBox struct{}

func (uikitBox) Show(ctx context.Context, n messagebox.Notice) error {
	raw, err := uikitapp.Ask(ctx, hostask.KindAlert, n.Title, n.Message)
	if err != nil {
		return err
	}
	return uikitapp.Accepted(raw)
}

func init() { driver.Register[messagebox.Driver](uikitFactory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = uikitFactory{}
	_ driver.Weighter                         = uikitFactory{}
)
