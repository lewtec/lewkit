// Package driver is a pluggable capability registry.
//
// Register a DriverFactory for an interface. The factory is both the
// enumerator (List) and the constructor (Handle.Open / New). List returns
// every compatible driver; Get opens the first. With and WithResult load
// the winner and call fn. LEWKIT_FORCE_DRIVER and LEWKIT_FORCE_<IFACE>_DRIVER
// pin an implementation for tests (weight 101); incompatible pins fall through.
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

// DriverFactory is one source of drivers for capability T. Name may
// change after CheckCompatibility (device name, display, …).
type DriverFactory[T any] interface {
	ID() string
	Name() string
	CheckCompatibility(ctx context.Context) error
	New(ctx context.Context) (T, error)
}

// Handle is one compatible driver. Open constructs it.
type Handle[T any] struct {
	ID     string
	Name   string
	Weight int
	open   func(context.Context) (T, error)
}

// Open constructs this driver.
func (h Handle[T]) Open(ctx context.Context) (T, error) {
	var zero T
	if h.open == nil {
		return zero, ErrUnavailable
	}
	return h.open(ctx)
}

// Weighter is optional on a factory. Get and Doctor use it when
// SetWeights has no entry for that id.
type Weighter interface {
	Weight() int
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

func factoryWeight(f any) int {
	w, ok := f.(Weighter)
	if !ok {
		return 0
	}
	return min(max(w.Weight(), 0), 100)
}

func effectiveWeight(weights map[string]int, driverID, ifaceName string, fallback int) int {
	if forced := forceDriverFromEnv(ifaceName); forced != "" && forced == driverID {
		return 101
	}
	if w, ok := weights[driverID]; ok {
		return w
	}
	return fallback
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
		Name:          factory.Name,
		Weight:        factoryWeight(factory),
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

type factorySet[T any] struct {
	typ       reflect.Type
	ifaceName string
	weights   map[string]int
	factories []DriverFactory[T]
}

func snapshot[T any]() (factorySet[T], error) {
	mu.RLock()
	defer mu.RUnlock()

	set := factorySet[T]{typ: reflect.TypeFor[T]()}
	if set.typ.Kind() != reflect.Interface {
		return set, ErrNotInterface
	}
	set.ifaceName = interfaceName(set.typ)
	set.weights = driverWeights[set.ifaceName]
	if byID, ok := Drivers[set.typ]; ok {
		for _, entry := range byID {
			f, ok := entry.(DriverFactory[T])
			if !ok {
				return set, fmt.Errorf("%w: %s", errCorrupt, set.ifaceName)
			}
			set.factories = append(set.factories, f)
		}
	}
	return set, nil
}

func (s factorySet[T]) sort() {
	slices.SortFunc(s.factories, func(a, b DriverFactory[T]) int {
		if c := cmp.Compare(effectiveWeight(s.weights, b.ID(), s.ifaceName, factoryWeight(b)), effectiveWeight(s.weights, a.ID(), s.ifaceName, factoryWeight(a))); c != 0 {
			return c
		}
		return cmp.Compare(a.ID(), b.ID())
	})
}

func (s factorySet[T]) weightOf(factory DriverFactory[T]) int {
	return effectiveWeight(s.weights, factory.ID(), s.ifaceName, factoryWeight(factory))
}

// List returns every compatible driver for T, highest weight first, without
// constructing them. Get opens the first Handle that New succeeds for.
func List[T any](ctx context.Context) ([]Handle[T], error) {
	set, err := snapshot[T]()
	if err != nil {
		return nil, err
	}
	if len(set.factories) == 0 {
		return nil, ErrNotFound
	}
	set.sort()

	var out []Handle[T]
	var report []string
	for _, factory := range set.factories {
		weight := set.weightOf(factory)
		if err := cachedCheck(factory.ID(), factory.CheckCompatibility, ctx); err != nil {
			report = append(report, fmt.Sprintf("[skip] %s (%s) weight=%d: %v", factory.ID(), factory.Name(), weight, err))
			continue
		}
		f := factory
		out = append(out, Handle[T]{
			ID:     f.ID(),
			Name:   f.Name(),
			Weight: weight,
			open:   f.New,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w for %s:\n%s", ErrUnavailable, set.typ.String(), strings.Join(report, "\n"))
	}
	return out, nil
}

// Get returns the highest-weight compatible implementation of T.
func Get[T any](ctx context.Context) (T, error) {
	var zero T
	handles, err := List[T](ctx)
	if err != nil {
		return zero, err
	}
	var report []string
	for _, handle := range handles {
		instance, err := handle.Open(ctx)
		if err != nil {
			report = append(report, fmt.Sprintf("[fail] %s (%s) weight=%d: %v", handle.ID, handle.Name, handle.Weight, err))
			continue
		}
		return instance, nil
	}
	return zero, fmt.Errorf("%w for %s:\n%s", ErrUnavailable, reflect.TypeFor[T]().String(), strings.Join(report, "\n"))
}
