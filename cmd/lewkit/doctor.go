package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type doctorCmd struct{}

func (doctorCmd) Description() string {
	return "list drivers (interface => implementation)"
}

func (*doctorCmd) Run(ctx context.Context) error {
	_, err := os.Stdout.WriteString(progress.Format(doctorNodes(driver.Doctor(ctx)), 0))
	return err
}

func doctorNodes(report []driver.InterfaceStatus) []taskgroup.Node {
	var nodes []taskgroup.Node
	var id taskgroup.ID
	next := func() taskgroup.ID {
		id++
		return id
	}
	for _, iface := range report {
		parent := next()
		selectedLabel := ""
		selectedWeight := 0
		available := 0
		for _, impl := range iface.Drivers {
			if impl.Selected {
				selectedLabel = driverLabel(impl)
				selectedWeight = impl.Weight
			}
			if impl.Available {
				available++
			}
		}
		message := "=> none"
		state := taskgroup.Failed
		if selectedLabel != "" {
			message = fmt.Sprintf("=> %s w=%d", selectedLabel, selectedWeight)
			state = taskgroup.Done
		}
		nodes = append(nodes, taskgroup.Node{
			ID:           parent,
			Name:         shortIface(iface.Name),
			Message:      message,
			State:        state,
			Pool:         taskgroup.Control,
			Current:      int64(available),
			Total:        int64(len(iface.Drivers)),
			LiveChildren: len(iface.Drivers),
		})
		for _, impl := range iface.Drivers {
			child := taskgroup.Node{
				ID:     next(),
				Parent: parent,
				Name:   impl.ID,
				Pool:   taskgroup.CPU,
			}
			detail := driverDetail(impl)
			switch {
			case impl.Selected:
				child.Message = detail + " selected"
				child.State = taskgroup.Done
			case impl.Available:
				child.Message = detail + " available"
				child.State = taskgroup.Done
			default:
				child.Message = detail
				if impl.Error != nil {
					child.Message = detail + " " + impl.Error.Error()
				}
				child.State = taskgroup.Failed
			}
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func driverLabel(impl driver.DriverStatus) string {
	if impl.ID == "" {
		return impl.Name
	}
	if impl.Name != "" && impl.Name != impl.ID {
		return impl.ID + ": " + impl.Name
	}
	return impl.ID
}

func driverDetail(impl driver.DriverStatus) string {
	weight := fmt.Sprintf("w=%d", impl.Weight)
	if impl.Name != "" && impl.Name != impl.ID {
		return impl.Name + " " + weight
	}
	return weight
}

func shortIface(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}
