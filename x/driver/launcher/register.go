package launcher

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
)

// Ask is a backend that can choose, prompt, and confirm.
type Ask interface {
	Chooser
	Prompter
	Confirmer
}

type askFactory[T Ask] struct {
	id     string
	weight int
	name   string
	check  func(context.Context) error
	open   func(context.Context) (T, error)
}

func (factory askFactory[T]) ID() string   { return factory.id }
func (factory askFactory[T]) Name() string { return factory.name }
func (factory askFactory[T]) Weight() int  { return factory.weight }

func (factory askFactory[T]) CheckCompatibility(ctx context.Context) error {
	return factory.check(ctx)
}

type askChooser[T Ask] struct{ askFactory[T] }

func (factory askChooser[T]) New(ctx context.Context) (Chooser, error) {
	opened, err := factory.open(ctx)
	return opened, err
}

type askPrompter[T Ask] struct{ askFactory[T] }

func (factory askPrompter[T]) New(ctx context.Context) (Prompter, error) {
	opened, err := factory.open(ctx)
	return opened, err
}

type askConfirmer[T Ask] struct{ askFactory[T] }

func (factory askConfirmer[T]) New(ctx context.Context) (Confirmer, error) {
	opened, err := factory.open(ctx)
	return opened, err
}

// RegisterAsk registers one backend for Choose, Prompt, and Confirm.
// The three names are what doctor shows. check and open are shared.
func RegisterAsk[T Ask](id string, weight int, chooseName, promptName, confirmName string, check func(context.Context) error, open func(context.Context) (T, error)) {
	gate := askFactory[T]{id: id, weight: weight, check: check, open: open}
	choose := gate
	choose.name = chooseName
	driver.Register[Chooser](askChooser[T]{choose})
	prompt := gate
	prompt.name = promptName
	driver.Register[Prompter](askPrompter[T]{prompt})
	confirm := gate
	confirm.name = confirmName
	driver.Register[Confirmer](askConfirmer[T]{confirm})
}
