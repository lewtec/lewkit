package main

import (
	"context"
	"iter"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/text/table"
)

type doctorCmd struct{}

func (doctorCmd) Description() string {
	return "list registered drivers"
}

func (*doctorCmd) Run(ctx context.Context) error {
	return cmd.Rows(ctx, os.Stdout, driverRows(driver.Doctor(ctx)), nil)
}

type driverRow struct {
	Interface string
	ID        string
	Name      string
	Weight    int
	Available bool
	Selected  bool
	Error     string
}

func (driverRow) Columns() []table.Column[driverRow] {
	return []table.Column[driverRow]{
		{Name: "id", Value: func(row driverRow) any { return row.ID }},
		{Name: "name", Value: func(row driverRow) any { return row.Name }},
		{Name: "interface", Value: func(row driverRow) any { return row.Interface }},
		{Name: "weight", Value: func(row driverRow) any { return row.Weight }},
		{Name: "available", Value: func(row driverRow) any { return row.Available }},
		{Name: "selected", Value: func(row driverRow) any { return row.Selected }},
		{Name: "error", Value: func(row driverRow) any { return row.Error }},
	}
}

func driverRows(report []driver.InterfaceStatus) iter.Seq[driverRow] {
	return func(yield func(driverRow) bool) {
		for _, iface := range report {
			for _, impl := range iface.Drivers {
				row := driverRow{
					Interface: iface.Name,
					ID:        impl.ID,
					Name:      impl.Name,
					Weight:    impl.Weight,
					Available: impl.Available,
					Selected:  impl.Selected,
				}
				if impl.Error != nil {
					row.Error = impl.Error.Error()
				}
				if !yield(row) {
					return
				}
			}
		}
	}
}
