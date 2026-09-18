package driver_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

var errBadID = errors.New("bad id")

type probe interface {
	ID() string
}

type probeFactory struct{}

func (probeFactory) ID() string                               { return "probe_test" }
func (probeFactory) Name() string                             { return "Probe" }
func (probeFactory) CheckCompatibility(context.Context) error { return nil }
func (probeFactory) New(context.Context) (probe, error)       { return probeImpl{}, nil }

type probeImpl struct{}

func (probeImpl) ID() string { return "probe_test" }

var registerProbe = sync.OnceFunc(func() {
	driver.Register[probe](probeFactory{})
})

func TestWithAndWithResult(t *testing.T) {
	registerProbe()
	t.Setenv("LEWKIT_FORCE_DRIVER", "probe_test")

	err := driver.With(t.Context(), func(p probe) error {
		if p.ID() != "probe_test" {
			return errBadID
		}
		return nil
	})
	require.NoError(t, err)

	id, err := driver.WithResult(t.Context(), func(p probe) (string, error) {
		return p.ID(), nil
	})
	require.NoError(t, err)
	require.Equal(t, "probe_test", id)
}

func TestWithMissing(t *testing.T) {
	type absent interface{ N() }
	err := driver.With(t.Context(), func(absent) error { return nil })
	require.ErrorIs(t, err, driver.ErrNotFound)

	_, err = driver.WithResult(t.Context(), func(absent) (int, error) { return 1, nil })
	require.ErrorIs(t, err, driver.ErrNotFound)
}
