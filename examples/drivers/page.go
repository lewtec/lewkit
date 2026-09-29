package main

import (
	"strings"

	"github.com/lewtec/lewkit/x/driver"
)

func driverMark(d driver.DriverStatus) (className, label string) {
	className, label = "badge-ghost", "unavailable"
	if d.Available {
		className, label = "", "available"
	}
	if d.Selected {
		className, label = "badge-success", "selected"
	}
	return className, label
}

func driverSlug(full string) string {
	rest := full
	if i := strings.LastIndex(full, "/"); i >= 0 {
		rest = full[i+1:]
	}
	return strings.TrimSuffix(rest, ".Driver")
}

func interfaceBySlug(report []driver.InterfaceStatus, slug string) (driver.InterfaceStatus, bool) {
	for _, iface := range report {
		if driverSlug(iface.Name) == slug {
			return iface, true
		}
	}
	return driver.InterfaceStatus{}, false
}

type row struct {
	Label string
	Value string
}

type field struct {
	Name        string
	Label       string
	Kind        string
	Value       string
	Placeholder string
	Min         string
	Max         string
	Step        string
}

type act struct {
	Op     string
	Label  string
	Fields []field
}

func (a act) hasFields() bool {
	for _, item := range a.Fields {
		if item.Kind != "hidden" {
			return true
		}
	}
	return false
}

type driverView struct {
	Slug     string
	Name     string
	Drivers  []driver.DriverStatus
	Rows     []row
	Acts     []act
	StateErr string
	Notice   string
	Err      string
}

func (v driverView) plainActs() []act {
	var out []act
	for _, item := range v.Acts {
		if !item.hasFields() {
			out = append(out, item)
		}
	}
	return out
}

func (v driverView) formActs() []act {
	var out []act
	for _, item := range v.Acts {
		if item.hasFields() {
			out = append(out, item)
		}
	}
	return out
}
