package driver

import (
	"cmp"
	"context"
	"maps"
	"reflect"
	"slices"
)

type offerMeta struct {
	ID     string
	Name   string
	Weight int
}

type doctorEntry struct {
	InterfaceType reflect.Type
	InterfaceName string
	FactoryType   reflect.Type
	DriverID      string
	Name          func() string
	Weight        int
	Check         func(context.Context) error
	Offers        func(context.Context) ([]offerMeta, error)
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
			ifaceStatus.Drivers = append(ifaceStatus.Drivers, d.statuses(ctx, weights)...)
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

func (d doctorEntry) statuses(ctx context.Context, weights map[string]int) []DriverStatus {
	checkErr := cachedCheck(d.DriverID, d.Check, ctx)
	factoryName := d.DriverID
	if d.Name != nil {
		factoryName = d.Name()
	}
	row := func(id, name string, fallback int, available bool, err error) DriverStatus {
		if id == "" {
			id = d.DriverID
		}
		if name == "" {
			name = factoryName
		}
		return DriverStatus{
			ID:          id,
			Name:        name,
			FactoryType: d.FactoryType,
			Weight:      effectiveWeight(weights, id, d.InterfaceName, fallback),
			Available:   available,
			Error:       err,
		}
	}
	if checkErr != nil || d.Offers == nil {
		return []DriverStatus{row(d.DriverID, factoryName, d.Weight, checkErr == nil, checkErr)}
	}
	offers, err := d.Offers(ctx)
	if err != nil || len(offers) == 0 {
		return []DriverStatus{row(d.DriverID, factoryName, d.Weight, false, err)}
	}
	out := make([]DriverStatus, 0, len(offers))
	for _, offer := range offers {
		out = append(out, row(offer.ID, offer.Name, offer.Weight, true, nil))
	}
	return out
}
