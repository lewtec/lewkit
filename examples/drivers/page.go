package main

import "github.com/lewtec/lewkit/x/driver"

func driverMark(d driver.DriverStatus) (className, label string) {
	className, label = "no", "unavailable"
	if d.Available {
		className, label = "ok", "available"
	}
	if d.Selected {
		label = "selected"
	}
	return className, label
}
