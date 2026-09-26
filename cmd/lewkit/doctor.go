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

type doctorCmd struct {
	cmd.Output `flatten:""`
}

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
		for _, interfaceStatus := range report {
			for _, driverStatus := range interfaceStatus.Drivers {
				row := driverRow{
					Interface: interfaceStatus.Name,
					ID:        driverStatus.ID,
					Name:      driverStatus.Name,
					Weight:    driverStatus.Weight,
					Available: driverStatus.Available,
					Selected:  driverStatus.Selected,
				}
				if driverStatus.Error != nil {
					row.Error = driverStatus.Error.Error()
				}
				if !yield(row) {
					return
				}
			}
		}
	}
}
