package main

import (
	"context"
	"iter"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
)

type doctorCmd struct{}

func (doctorCmd) Description() string {
	return "list registered drivers"
}

func (*doctorCmd) Run(ctx context.Context) error {
	return cmd.Rows(ctx, os.Stdout, driverRows(driver.Doctor(ctx)))
}

type driverRow struct {
	Interface string `json:"interface" table:",order=2"`
	ID        string `json:"id" table:",order=0"`
	Name      string `json:"name" table:",order=1"`
	Weight    int    `json:"weight" table:",order=3"`
	Available bool   `json:"available" table:",order=4"`
	Selected  bool   `json:"selected" table:",order=5"`
	Error     string `json:"error,omitempty" table:",order=6"`
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
