package launcher

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

var errAskProbe = errors.New("ask probe")

type registeredAskBackend struct{}

func (registeredAskBackend) Choose(context.Context, ChooseOptions) (*Item, error) {
	return nil, nil
}

func (registeredAskBackend) Prompt(context.Context, string) (string, error) { return "", nil }

func (registeredAskBackend) Confirm(context.Context, string) (bool, error) { return false, nil }

var registerAskOnce sync.Once

func TestRegisterAsk(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_DRIVER", "")
	registerAskOnce.Do(func() {
		RegisterAsk("ask_probe", 7, "Probe (Choose)", "Probe (Prompt)", "Probe (Confirm)",
			func(context.Context) error { return errAskProbe },
			func(context.Context) (registeredAskBackend, error) { return registeredAskBackend{}, nil },
		)
	})
	want := map[string]string{
		"github.com/lewtec/lewkit/x/driver/launcher.Chooser":   "Probe (Choose)",
		"github.com/lewtec/lewkit/x/driver/launcher.Prompter":  "Probe (Prompt)",
		"github.com/lewtec/lewkit/x/driver/launcher.Confirmer": "Probe (Confirm)",
	}
	found := map[string]bool{}
	for _, iface := range driver.Doctor(t.Context()) {
		name, ok := want[iface.Name]
		if !ok {
			continue
		}
		for _, row := range iface.Drivers {
			if row.ID != "ask_probe" {
				continue
			}
			require.Equal(t, name, row.Name)
			require.Equal(t, 7, row.Weight)
			require.ErrorIs(t, row.Error, errAskProbe)
			require.False(t, row.Available)
			found[iface.Name] = true
		}
	}
	require.Len(t, found, len(want))
}
