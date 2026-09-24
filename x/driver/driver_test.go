package driver_test

import (
	"context"
	"errors"
	"iter"
	"runtime"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

var errInitFailed = errors.New("init failed")

func TestForGOOS(t *testing.T) {
	require.NoError(t, driver.ForGOOS(runtime.GOOS))
	other := "plan9"
	if runtime.GOOS == other {
		other = "windows"
	}
	err := driver.ForGOOS(other)
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.Contains(t, err.Error(), "not "+other)
}

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

func TestListCompatible(t *testing.T) {
	registerPickers()
	iface := pickerIfaceName()
	setAllWeights(t, iface, map[string]int{
		"pick_high":   80,
		"pick_mid":    90,
		"pick_low":    10,
		"pick_broken": 70,
	})

	handles, err := driver.List[picker](t.Context())
	require.NoError(t, err)
	var ids []string
	var names []string
	for _, handle := range handles {
		ids = append(ids, handle.ID)
		names = append(names, handle.Name)
	}
	require.Equal(t, []string{"pick_high", "pick_broken", "pick_low"}, ids)
	require.Equal(t, []string{"High", "Broken", "Low"}, names)

	got, err := handles[0].Open(t.Context())
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

type namedProbe interface{ ID() string }

type namedProbeFactory struct {
	ready bool
}

func (f *namedProbeFactory) ID() string { return "named_probe" }
func (f *namedProbeFactory) Name() string {
	if f.ready {
		return "Apple M5"
	}
	return "Probe"
}
func (f *namedProbeFactory) CheckCompatibility(context.Context) error {
	f.ready = true
	return nil
}
func (f *namedProbeFactory) New(context.Context) (namedProbe, error) {
	return pickerImpl{id: f.ID()}, nil
}

var namedProbeFac = &namedProbeFactory{}

var registerNamedProbe = sync.OnceFunc(func() {
	driver.Register[namedProbe](namedProbeFac)
})

func TestDoctorUsesNameAfterCheck(t *testing.T) {
	registerNamedProbe()
	var found *driver.DriverStatus
	for _, st := range driver.Doctor(t.Context()) {
		for i := range st.Drivers {
			if st.Drivers[i].ID == "named_probe" {
				found = &st.Drivers[i]
			}
		}
	}
	require.NotNil(t, found)
	require.Equal(t, "Apple M5", found.Name)
	require.True(t, found.Available)
}

func TestListUsesNameAfterCheck(t *testing.T) {
	registerNamedProbe()
	handles, err := driver.List[namedProbe](t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, handles)
	require.Equal(t, "named_probe", handles[0].ID)
	require.Equal(t, "Apple M5", handles[0].Name)
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

type ranked interface{ ID() string }

type rankedFactory struct {
	id     string
	weight int
	check  error
}

func (f rankedFactory) ID() string   { return f.id }
func (f rankedFactory) Name() string { return f.id }
func (f rankedFactory) Weight() int  { return f.weight }
func (f rankedFactory) CheckCompatibility(context.Context) error {
	return f.check
}
func (f rankedFactory) New(context.Context) (ranked, error) {
	return pickerImpl{id: f.id}, nil
}

var registerRanked = sync.OnceFunc(func() {
	driver.Register[ranked](rankedFactory{id: "rank_light", weight: 10})
	driver.Register[ranked](rankedFactory{id: "rank_heavy", weight: 80})
	driver.Register[ranked](rankedFactory{id: "rank_blocked", weight: 90, check: driver.ErrIncompatible})
})

func TestGetUsesFactoryWeight(t *testing.T) {
	registerRanked()
	driver.ResetWeights()
	got, err := driver.Get[ranked](t.Context())
	require.NoError(t, err)
	require.Equal(t, "rank_heavy", got.ID())
}

func TestDoctorReportsFactoryWeight(t *testing.T) {
	registerRanked()
	driver.ResetWeights()
	var found *driver.InterfaceStatus
	for _, st := range driver.Doctor(t.Context()) {
		if len(st.Drivers) >= 3 && st.Drivers[0].ID == "rank_blocked" {
			found = &st
			break
		}
	}
	require.NotNil(t, found)
	byID := map[string]driver.DriverStatus{}
	for _, d := range found.Drivers {
		byID[d.ID] = d
	}
	require.Equal(t, 90, byID["rank_blocked"].Weight)
	require.Equal(t, 80, byID["rank_heavy"].Weight)
	require.Equal(t, 10, byID["rank_light"].Weight)
	require.True(t, byID["rank_heavy"].Selected)
	require.False(t, byID["rank_blocked"].Selected)
}

func TestSetWeightsOverridesFactoryWeight(t *testing.T) {
	registerRanked()
	iface := ""
	for name, ids := range driver.RegisteredWeightShape() {
		for _, id := range ids {
			if id == "rank_heavy" {
				iface = name
			}
		}
	}
	require.NotEmpty(t, iface)
	setAllWeights(t, iface, map[string]int{
		"rank_light":   100,
		"rank_heavy":   1,
		"rank_blocked": 0,
	})
	got, err := driver.Get[ranked](t.Context())
	require.NoError(t, err)
	require.Equal(t, "rank_light", got.ID())
}

func TestRegisteredWeightShape(t *testing.T) {
	registerPickers()
	shape := driver.RegisteredWeightShape()
	iface := pickerIfaceName()
	require.Equal(t, []string{"pick_broken", "pick_high", "pick_low", "pick_mid"}, shape[iface])
}

type gpus interface{ ID() string }

type gpuFactory struct{}

func (gpuFactory) ID() string                               { return "gpu_all" }
func (gpuFactory) Name() string                             { return "GPU" }
func (gpuFactory) Weight() int                              { return 40 }
func (gpuFactory) CheckCompatibility(context.Context) error { return nil }
func (gpuFactory) New(context.Context) (gpus, error) {
	return pickerImpl{id: "gpu_all:amd"}, nil
}
func (gpuFactory) Offers(context.Context) iter.Seq2[driver.Offer[gpus], error] {
	return func(yield func(driver.Offer[gpus], error) bool) {
		for _, o := range []driver.Offer[gpus]{
			{ID: "gpu_all:amd", Name: "Renoir", Weight: 40, New: func(context.Context) (gpus, error) {
				return pickerImpl{id: "gpu_all:amd"}, nil
			}},
			{ID: "gpu_all:nvidia", Name: "RTX 3060", Weight: 40, New: func(context.Context) (gpus, error) {
				return pickerImpl{id: "gpu_all:nvidia"}, nil
			}},
			{ID: "gpu_all:llvmpipe", Name: "llvmpipe", Weight: 0, New: func(context.Context) (gpus, error) {
				return pickerImpl{id: "gpu_all:llvmpipe"}, nil
			}},
		} {
			if !yield(o, nil) {
				return
			}
		}
	}
}

var registerGPUs = sync.OnceFunc(func() {
	driver.Register[gpus](gpuFactory{})
})

func gpuHandles(t *testing.T) []driver.Handle[gpus] {
	t.Helper()
	registerGPUs()
	driver.ResetWeights()
	handles, err := driver.List[gpus](t.Context())
	require.NoError(t, err)
	return handles
}

func TestListOffersEveryDevice(t *testing.T) {
	handles := gpuHandles(t)
	var ids, names []string
	for _, handle := range handles {
		ids = append(ids, handle.ID)
		names = append(names, handle.Name)
	}
	require.Equal(t, []string{"gpu_all:amd", "gpu_all:nvidia", "gpu_all:llvmpipe"}, ids)
	require.Equal(t, []string{"Renoir", "RTX 3060", "llvmpipe"}, names)
	require.Equal(t, 40, handles[0].Weight)
	require.Equal(t, 40, handles[1].Weight)
	require.Equal(t, 0, handles[2].Weight)
	got, err := driver.Get[gpus](t.Context())
	require.NoError(t, err)
	require.Equal(t, "gpu_all:amd", got.ID())
}

func TestGetForceOffer(t *testing.T) {
	registerGPUs()
	driver.ResetWeights()
	t.Setenv("LEWKIT_FORCE_DRIVER", "gpu_all:nvidia")
	got, err := driver.Get[gpus](t.Context())
	require.NoError(t, err)
	require.Equal(t, "gpu_all:nvidia", got.ID())
}

func TestGetVendorWeight(t *testing.T) {
	registerGPUs()
	iface := ""
	for name, ids := range driver.RegisteredWeightShape() {
		for _, id := range ids {
			if id == "gpu_all" {
				iface = name
			}
		}
	}
	require.NotEmpty(t, iface)
	setAllWeights(t, iface, map[string]int{
		"gpu_all":          40,
		"gpu_all:nvidia":   90,
		"gpu_all:llvmpipe": 0,
	})
	got, err := driver.Get[gpus](t.Context())
	require.NoError(t, err)
	require.Equal(t, "gpu_all:nvidia", got.ID())
}

func TestDoctorListsOffers(t *testing.T) {
	registerGPUs()
	driver.ResetWeights()
	var found *driver.InterfaceStatus
	for _, st := range driver.Doctor(t.Context()) {
		for _, d := range st.Drivers {
			if d.ID == "gpu_all:amd" || d.ID == "gpu_all:nvidia" {
				found = &st
				break
			}
		}
	}
	require.NotNil(t, found)
	byID := map[string]driver.DriverStatus{}
	for _, d := range found.Drivers {
		byID[d.ID] = d
	}
	require.Equal(t, "Renoir", byID["gpu_all:amd"].Name)
	require.Equal(t, "RTX 3060", byID["gpu_all:nvidia"].Name)
	require.Equal(t, "llvmpipe", byID["gpu_all:llvmpipe"].Name)
	require.Equal(t, 40, byID["gpu_all:amd"].Weight)
	require.Equal(t, 0, byID["gpu_all:llvmpipe"].Weight)
	require.True(t, byID["gpu_all:amd"].Selected)
	require.False(t, byID["gpu_all:nvidia"].Selected)
	require.False(t, byID["gpu_all:llvmpipe"].Selected)
	require.True(t, byID["gpu_all:amd"].Available)
	require.True(t, byID["gpu_all:nvidia"].Available)
	require.True(t, byID["gpu_all:llvmpipe"].Available)
}
