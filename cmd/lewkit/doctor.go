package main

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/text/table"
)

type doctorCmd struct {
	cmd.Output `flatten:""`
}

func (doctorCmd) Description() string {
	return "list registered drivers"
}

func (command *doctorCmd) Run(ctx context.Context) error {
	report := driver.Doctor(ctx)
	write := func(context.Context) error {
		if command.Columns.Value() != "" {
			return cmd.Rows(ctx, os.Stdout, implementationRows(report), driverLayout)
		}
		return cmd.Rows(ctx, os.Stdout, doctorLines(report), lineLayout)
	}
	// The progress view stops after Run returns. Print once the terminal is back.
	if taskgroup.FromContext(ctx) != nil {
		entry.After(write)
		return nil
	}
	return write(ctx)
}

// driverRow is one implementation. --columns selects these fields.
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

// doctorLine is the default table. Mark is the status glyph.
// Driver is the interface, or the implementation id with a two-space margin.
// Detail collapses name, weight, and error.
type doctorLine struct {
	Mark   string `json:"mark"`
	Driver string `json:"driver"`
	Detail string `json:"detail"`
}

type lineSpec struct {
	Mark   table.Field[string]
	Driver table.Field[string]
	Detail table.Field[string]
}

var lineLayout = table.Must(table.Make[doctorLine](lineSpec{}))

func implementationRows(report []driver.InterfaceStatus) iter.Seq[driverRow] {
	return func(yield func(driverRow) bool) {
		for _, interfaceStatus := range report {
			for _, driverStatus := range interfaceStatus.Drivers {
				if !yield(implementationRow(interfaceStatus.Name, driverStatus)) {
					return
				}
			}
		}
	}
}

func doctorLines(report []driver.InterfaceStatus) iter.Seq[doctorLine] {
	return func(yield func(doctorLine) bool) {
		for _, interfaceStatus := range report {
			if !yield(doctorLine{Driver: shortInterface(interfaceStatus.Name)}) {
				return
			}
			for _, driverStatus := range interfaceStatus.Drivers {
				line := doctorLine{
					Mark:   statusMark(driverStatus),
					Driver: "  " + driverStatus.ID,
					Detail: implementationDetail(driverStatus),
				}
				if !yield(line) {
					return
				}
			}
		}
	}
}

func implementationRow(interfaceName string, driverStatus driver.DriverStatus) driverRow {
	row := driverRow{
		Interface: interfaceName,
		ID:        driverStatus.ID,
		Name:      driverStatus.Name,
		Weight:    driverStatus.Weight,
		Available: driverStatus.Available,
		Selected:  driverStatus.Selected,
	}
	if driverStatus.Error != nil {
		row.Error = driverStatus.Error.Error()
	}
	return row
}

func shortInterface(name string) string {
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		return name[slash+1:]
	}
	return name
}

func statusMark(driverStatus driver.DriverStatus) string {
	switch {
	case driverStatus.Selected:
		return "✓"
	case driverStatus.Available:
		return "·"
	default:
		return "✗"
	}
}

func implementationDetail(driverStatus driver.DriverStatus) string {
	var parts []string
	if driverStatus.Name != "" && driverStatus.Name != driverStatus.ID {
		parts = append(parts, driverStatus.Name)
	}
	parts = append(parts, fmt.Sprintf("w=%d", driverStatus.Weight))
	if driverStatus.Error != nil {
		parts = append(parts, driverStatus.Error.Error())
	}
	return strings.Join(parts, "  ")
}
