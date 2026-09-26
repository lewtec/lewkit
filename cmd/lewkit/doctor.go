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
	return cmd.Rows(ctx, os.Stdout, driverRows(driver.Doctor(ctx)), driverLayout)
}

type driverRow struct {
	Interface string `json:"interface"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Weight    int    `json:"weight"`
	Available bool   `json:"available"`
	Selected  bool   `json:"selected"`
	Error     string `json:"error"`
}

type driverSpec struct {
	ID        table.Field[string]
	Name      table.Field[string]
	Interface table.Field[string]
	Weight    table.Field[int]
	Available table.Field[bool]
	Selected  table.Field[bool]
	Error     table.Field[string]
}

var driverLayout = table.Must(table.Make[driverRow](driverSpec{}))

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
