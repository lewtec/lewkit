// Package driver is a pluggable capability registry.
//
// Register a DriverFactory for an interface. Get selects by weight and
// CheckCompatibility. With and WithResult load the winner and call fn.
// LEWKIT_FORCE_DRIVER and LEWKIT_FORCE_<IFACE>_DRIVER pin an implementation
// for tests (weight 101); incompatible pins fall through.
package driver

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
)

var (
	// ErrIncompatible means the implementation cannot run here.
	ErrIncompatible = errors.New("component is incompatible")
	// ErrNotInterface means Get or Register was used with a non-interface type.
	ErrNotInterface = errors.New("driver is not an interface")
	// ErrNotFound means no factory is registered for the interface.
	ErrNotFound = errors.New("driver not found")
	// ErrInvalidWeight means a configured weight is outside 0–100.
	ErrInvalidWeight = errors.New("invalid driver weight")
	// ErrMissingWeight means SetWeights omitted a registered factory.
	ErrMissingWeight = errors.New("missing driver weight")
	// ErrUnavailable means every registered factory failed compatibility or New.
	ErrUnavailable = errors.New("no available driver")
	errCorrupt     = errors.New("driver registry corrupt")
)

// DriverFactory constructs one implementation of capability T.
type DriverFactory[T any] interface {
	ID() string
	Name() string
	CheckCompatibility(ctx context.Context) error
	New(ctx context.Context) (T, error)
}

type validationResult struct {
	once sync.Once
	err  error
}

var (
	mu sync.RWMutex
	// Drivers is the process-wide factory table, keyed by capability type then ID.
	Drivers         = map[reflect.Type]map[string]any{}
	driverWeights   = map[string]map[string]int{}
	doctorList      = []doctorEntry{}
	validationCache sync.Map // driver ID -> *validationResult
)

func cachedCheck(id string, check func(context.Context) error, ctx context.Context) error {
	val, alreadyPresent := validationCache.LoadOrStore(id, &validationResult{})
	vr, ok := val.(*validationResult)
	if !ok {
		return fmt.Errorf("%w: driver %q", errCorrupt, id)
	}
	vr.once.Do(func() { vr.err = check(ctx) })
	if alreadyPresent {
		return vr.err
	}
	return vr.err
}

// SetWeights configures driver priorities. Weights must be between 0 and 100.
// Every registered factory must have an entry.
func SetWeights(w map[string]map[string]int) error {
	mu.Lock()
	defer mu.Unlock()

	for iface, drivers := range w {
		for id, weight := range drivers {
			if weight < 0 || weight > 100 {
				return fmt.Errorf("%w: %d for driver %q in interface %q", ErrInvalidWeight, weight, id, iface)
			}
		}
	}
	for ifaceType, factories := range Drivers {
		ifaceName := interfaceName(ifaceType)
		configured, ok := w[ifaceName]
		if !ok {
			return fmt.Errorf("%w for interface %q", ErrMissingWeight, ifaceName)
		}
		for id := range factories {
			if _, ok := configured[id]; !ok {
				return fmt.Errorf("%w for %q in interface %q", ErrMissingWeight, id, ifaceName)
			}
		}
	}
	cloned := make(map[string]map[string]int, len(w))
	for iface, drivers := range w {
		cloned[iface] = maps.Clone(drivers)
	}
	driverWeights = cloned
	return nil
}

func forceDriverFromEnv(ifaceName string) string {
	if v := os.Getenv("LEWKIT_FORCE_DRIVER"); v != "" {
		return v
	}
	key := ifaceName
	if _, after, ok := strings.CutLast(key, "/"); ok {
		key = after
	}
	key = strings.TrimSuffix(key, ".Driver")
	key = strings.ReplaceAll(key, ".", "_")
	key = "LEWKIT_FORCE_" + strings.ToUpper(key) + "_DRIVER"
	if v := os.Getenv(key); v != "" {
		return v
	}
	return ""
}

func effectiveWeight(weights map[string]int, driverID, ifaceName string) int {
	if forced := forceDriverFromEnv(ifaceName); forced != "" && forced == driverID {
		return 101
	}
	return weights[driverID]
}

// Register adds a factory for capability T. T must be an interface.
// Panics on an empty ID or a duplicate ID for T.
func Register[T any](factory DriverFactory[T]) {
	mu.Lock()
	defer mu.Unlock()

	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Interface {
		panic("driver " + t.String() + " is not an interface")
	}
	id := factory.ID()
	if id == "" {
		panic("driver for " + t.String() + " registered with empty ID")
	}

	if _, ok := Drivers[t]; !ok {
		Drivers[t] = make(map[string]any)
	}
	if _, ok := Drivers[t][id]; ok {
		panic("driver ID " + id + " already registered for interface " + t.String())
	}
	Drivers[t][id] = factory

	doctorList = append(doctorList, doctorEntry{
		InterfaceType: t,
		InterfaceName: interfaceName(t),
		FactoryType:   reflect.TypeOf(factory),
		DriverID:      id,
		DriverName:    factory.Name(),
		Check:         factory.CheckCompatibility,
	})
}

func interfaceName(t reflect.Type) string {
	if t.PkgPath() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	return t.String()
}

// RegisteredWeightShape is interface name → sorted factory IDs.
func RegisteredWeightShape() map[string][]string {
	mu.RLock()
	defer mu.RUnlock()

	shape := make(map[string][]string, len(Drivers))
	for ifaceType, factories := range Drivers {
		shape[interfaceName(ifaceType)] = slices.Sorted(maps.Keys(factories))
	}
	return shape
}

// Get returns the highest-weight compatible implementation of T.
func Get[T any](ctx context.Context) (T, error) {
	mu.RLock()
	t := reflect.TypeFor[T]()
	if t.Kind() != reflect.Interface {
		mu.RUnlock()
		var zero T
		return zero, ErrNotInterface
	}

	ifaceName := interfaceName(t)
	weights := driverWeights[ifaceName]

	var factories []DriverFactory[T]
	if byID, ok := Drivers[t]; ok {
		for _, entry := range byID {
			f, ok := entry.(DriverFactory[T])
			if !ok {
				mu.RUnlock()
				var zero T
				return zero, fmt.Errorf("%w: %s", errCorrupt, ifaceName)
			}
			factories = append(factories, f)
		}
	}
	mu.RUnlock()

	var zero T
	if len(factories) == 0 {
		return zero, ErrNotFound
	}

	slices.SortFunc(factories, func(a, b DriverFactory[T]) int {
		if c := cmp.Compare(effectiveWeight(weights, b.ID(), ifaceName), effectiveWeight(weights, a.ID(), ifaceName)); c != 0 {
			return c
		}
		return cmp.Compare(a.ID(), b.ID())
	})

	var report []string
	for _, factory := range factories {
		weight := effectiveWeight(weights, factory.ID(), ifaceName)
		if err := cachedCheck(factory.ID(), factory.CheckCompatibility, ctx); err != nil {
			report = append(report, fmt.Sprintf("[skip] %s (%s) weight=%d: %v", factory.ID(), factory.Name(), weight, err))
			continue
		}
		instance, err := factory.New(ctx)
		if err != nil {
			report = append(report, fmt.Sprintf("[fail] %s (%s) weight=%d: %v", factory.ID(), factory.Name(), weight, err))
			continue
		}
		return instance, nil
	}

	return zero, fmt.Errorf("%w for %s:\n%s", ErrUnavailable, t.String(), strings.Join(report, "\n"))
}
