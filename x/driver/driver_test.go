package driver_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

var errInitFailed = errors.New("init failed")

type picker interface {
	ID() string
}

type pickerFactory struct {
	id    string
	name  string
	check error
	new   error
}

func (f pickerFactory) ID() string   { return f.id }
func (f pickerFactory) Name() string { return f.name }
func (f pickerFactory) CheckCompatibility(context.Context) error {
	return f.check
}
func (f pickerFactory) New(context.Context) (picker, error) {
	if f.new != nil {
		return nil, f.new
	}
	return pickerImpl{id: f.id}, nil
}

type pickerImpl struct{ id string }

func (p pickerImpl) ID() string { return p.id }

var registerPickers = sync.OnceFunc(func() {
	driver.Register[picker](pickerFactory{id: "pick_high", name: "High"})
	driver.Register[picker](pickerFactory{id: "pick_mid", name: "Mid", check: driver.ErrIncompatible})
	driver.Register[picker](pickerFactory{id: "pick_low", name: "Low"})
	driver.Register[picker](pickerFactory{id: "pick_broken", name: "Broken", new: errInitFailed})
})

func pickerIfaceName() string {
	for name, ids := range driver.RegisteredWeightShape() {
		for _, id := range ids {
			if id == "pick_high" {
				return name
			}
		}
	}
	return ""
}

func setAllWeights(t *testing.T, iface string, idWeights map[string]int) {
	t.Helper()
	w := map[string]map[string]int{}
	for name, ids := range driver.RegisteredWeightShape() {
		inner := map[string]int{}
		for _, id := range ids {
			inner[id] = 0
		}
		w[name] = inner
	}
	if iface != "" {
		if _, ok := w[iface]; !ok {
			w[iface] = map[string]int{}
		}
		for id, wt := range idWeights {
			w[iface][id] = wt
		}
	}
	require.NoError(t, driver.SetWeights(w))
}

func TestGetPicksHighestCompatible(t *testing.T) {
	registerPickers()
	iface := pickerIfaceName()
	require.NotEmpty(t, iface)
	setAllWeights(t, iface, map[string]int{
		"pick_high":   80,
		"pick_mid":    90,
		"pick_low":    10,
		"pick_broken": 70,
	})

	got, err := driver.Get[picker](t.Context())
	require.NoError(t, err)
	require.Equal(t, "pick_high", got.ID())
}

func TestGetSkipsInitFailure(t *testing.T) {
	registerPickers()
	iface := pickerIfaceName()
	setAllWeights(t, iface, map[string]int{
		"pick_high":   0,
		"pick_mid":    0,
		"pick_low":    5,
		"pick_broken": 100,
	})

	got, err := driver.Get[picker](t.Context())
	require.NoError(t, err)
	require.Equal(t, "pick_low", got.ID())
}

func TestGetForceEnv(t *testing.T) {
	registerPickers()
	iface := pickerIfaceName()
	setAllWeights(t, iface, map[string]int{
		"pick_high":   80,
		"pick_mid":    90,
		"pick_low":    10,
		"pick_broken": 70,
	})
	t.Setenv("LEWKIT_FORCE_DRIVER", "pick_low")

	got, err := driver.Get[picker](t.Context())
	require.NoError(t, err)
	require.Equal(t, "pick_low", got.ID())
}

func TestGetForceIncompatibleFallsThrough(t *testing.T) {
	registerPickers()
	iface := pickerIfaceName()
	setAllWeights(t, iface, map[string]int{
		"pick_high":   80,
		"pick_mid":    90,
		"pick_low":    10,
		"pick_broken": 70,
	})
	t.Setenv("LEWKIT_FORCE_DRIVER", "pick_mid")

	got, err := driver.Get[picker](t.Context())
	require.NoError(t, err)
	require.Equal(t, "pick_high", got.ID())
}

type dead interface{ Dead() }

type deadFactory struct{}

func (deadFactory) ID() string                               { return "dead_only" }
func (deadFactory) Name() string                             { return "Dead" }
func (deadFactory) CheckCompatibility(context.Context) error { return driver.ErrIncompatible }
func (deadFactory) New(context.Context) (dead, error)        { return nil, nil }

var registerDead = sync.OnceFunc(func() {
	driver.Register[dead](deadFactory{})
})

func TestGetUnavailable(t *testing.T) {
	registerDead()
	_, err := driver.Get[dead](t.Context())
	require.ErrorIs(t, err, driver.ErrUnavailable)
}

func TestGetNotFound(t *testing.T) {
	type absent interface{ N() }
	_, err := driver.Get[absent](t.Context())
	require.ErrorIs(t, err, driver.ErrNotFound)
}

func TestGetNotInterface(t *testing.T) {
	_, err := driver.Get[int](t.Context())
	require.ErrorIs(t, err, driver.ErrNotInterface)
}

func TestSetWeightsRejectsRange(t *testing.T) {
	err := driver.SetWeights(map[string]map[string]int{
		"x": {"a": 101},
	})
	require.ErrorIs(t, err, driver.ErrInvalidWeight)
}

func TestSetWeightsRequiresRegistered(t *testing.T) {
	registerPickers()
	err := driver.SetWeights(map[string]map[string]int{})
	require.ErrorIs(t, err, driver.ErrMissingWeight)
}

func TestDoctorMarksSelected(t *testing.T) {
	registerPickers()
	iface := pickerIfaceName()
	setAllWeights(t, iface, map[string]int{
		"pick_high":   80,
		"pick_mid":    90,
		"pick_low":    10,
		"pick_broken": 70,
	})

	var found *driver.InterfaceStatus
	for _, st := range driver.Doctor(t.Context()) {
		if st.Name == iface {
			found = &st
			break
		}
	}
	require.NotNil(t, found)
	require.GreaterOrEqual(t, len(found.Drivers), 4)

	var selected []string
	for _, d := range found.Drivers {
		if d.Selected {
			selected = append(selected, d.ID)
		}
	}
	require.Equal(t, []string{"pick_high"}, selected)
}

func TestRegisteredWeightShape(t *testing.T) {
	registerPickers()
	shape := driver.RegisteredWeightShape()
	iface := pickerIfaceName()
	require.Equal(t, []string{"pick_broken", "pick_high", "pick_low", "pick_mid"}, shape[iface])
}
