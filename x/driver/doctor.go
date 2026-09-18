package driver

import (
	"cmp"
	"context"
	"maps"
	"reflect"
	"slices"
)

type doctorEntry struct {
	InterfaceType reflect.Type
	InterfaceName string
	FactoryType   reflect.Type
	DriverID      string
	DriverName    string
	Check         func(context.Context) error
}

// DriverStatus is one factory in a Doctor report.
type DriverStatus struct {
	ID          string
	Name        string
	FactoryType reflect.Type
	Weight      int
	Available   bool
	Selected    bool
	Error       error
}

// InterfaceStatus is one capability in a Doctor report.
type InterfaceStatus struct {
	Name    string
	Drivers []DriverStatus
}

// Doctor reports every registered factory, marked available/selected.
func Doctor(ctx context.Context) []InterfaceStatus {
	mu.RLock()
	byType := make(map[reflect.Type][]doctorEntry)
	for _, d := range doctorList {
		byType[d.InterfaceType] = append(byType[d.InterfaceType], d)
	}
	weightsSnapshot := make(map[string]map[string]int, len(driverWeights))
	for name, w := range driverWeights {
		weightsSnapshot[name] = maps.Clone(w)
	}
	mu.RUnlock()

	types := slices.SortedFunc(maps.Keys(byType), func(a, b reflect.Type) int {
		return cmp.Compare(interfaceName(a), interfaceName(b))
	})

	var result []InterfaceStatus
	for _, t := range types {
		ifaceName := interfaceName(t)
		weights := weightsSnapshot[ifaceName]
		ifaceStatus := InterfaceStatus{Name: ifaceName}
		for _, d := range byType[t] {
			err := cachedCheck(d.DriverID, d.Check, ctx)
			ifaceStatus.Drivers = append(ifaceStatus.Drivers, DriverStatus{
				ID:          d.DriverID,
				Name:        d.DriverName,
				FactoryType: d.FactoryType,
				Weight:      effectiveWeight(weights, d.DriverID, ifaceName),
				Available:   err == nil,
				Error:       err,
			})
		}
		slices.SortFunc(ifaceStatus.Drivers, func(a, b DriverStatus) int {
			if c := cmp.Compare(b.Weight, a.Weight); c != 0 {
				return c
			}
			return cmp.Compare(a.ID, b.ID)
		})
		for i := range ifaceStatus.Drivers {
			if ifaceStatus.Drivers[i].Available {
				ifaceStatus.Drivers[i].Selected = true
				break
			}
		}
		result = append(result, ifaceStatus)
	}
	return result
}
