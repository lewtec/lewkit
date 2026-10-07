package darwin

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/battery"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type backend struct{}

type reading struct {
	status   battery.Status
	level    int
	levelErr error
}

func (backend) BatteryStatus(ctx context.Context) (battery.Status, error) {
	got, err := read(ctx)
	if err != nil {
		return battery.Unknown, err
	}
	return got.status, nil
}

func (backend) BatteryLevel(ctx context.Context) (int, error) {
	got, err := read(ctx)
	if err != nil {
		return 0, err
	}
	return got.level, got.levelErr
}

func read(ctx context.Context) (reading, error) {
	if err := ctx.Err(); err != nil {
		return reading{}, err
	}
	out, err := execdriver.Output(ctx, execdriver.MustCommand(ctx, ioregBin, "-rn", "AppleSmartBattery", "-d", "1"))
	if err != nil {
		return reading{}, fmt.Errorf("ioreg: %w", err)
	}
	return parseIOReg(string(out))
}

// Property lines are `"Key" = value`. Nested blobs use `=` with no spaces.
func properties(text string) map[string]string {
	got := map[string]string{}
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, `" = `)
		if !ok || !strings.HasPrefix(key, `"`) {
			continue
		}
		key = strings.TrimPrefix(key, `"`)
		switch key {
		case "BatteryInstalled", "IsCharging", "FullyCharged", "ExternalConnected", "CurrentCapacity", "MaxCapacity":
			if _, seen := got[key]; !seen {
				got[key] = strings.TrimSpace(val)
			}
		}
	}
	return got
}

func parseIOReg(text string) (reading, error) {
	props := properties(text)
	if !present(props) {
		return reading{}, battery.ErrNoBattery
	}
	level, levelErr := levelOf(props)
	return reading{status: statusOf(props), level: level, levelErr: levelErr}, nil
}

func present(props map[string]string) bool {
	if no(props["BatteryInstalled"]) {
		return false
	}
	if yes(props["BatteryInstalled"]) {
		return true
	}
	return props["CurrentCapacity"] != "" || props["IsCharging"] != "" || props["FullyCharged"] != ""
}

func statusOf(props map[string]string) battery.Status {
	if yes(props["FullyCharged"]) {
		return battery.Full
	}
	if yes(props["IsCharging"]) {
		return battery.Charging
	}
	if no(props["ExternalConnected"]) {
		return battery.Discharging
	}
	if yes(props["ExternalConnected"]) {
		return battery.Status("Not charging")
	}
	return battery.Unknown
}

func levelOf(props map[string]string) (int, error) {
	raw, ok := props["CurrentCapacity"]
	if !ok {
		return 0, battery.ErrUnknownLevel
	}
	now, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("current capacity: %w", err)
	}
	rawMax, ok := props["MaxCapacity"]
	if !ok {
		if now < 0 || now > 100 {
			return 0, battery.ErrUnknownLevel
		}
		return int(now), nil
	}
	full, err := strconv.ParseInt(rawMax, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("max capacity: %w", err)
	}
	return percentOf(now, full)
}

func percentOf(now, full int64) (int, error) {
	if now < 0 || full <= 0 {
		return 0, battery.ErrUnknownLevel
	}
	if now > full {
		now = full
	}
	return int((now*100 + full/2) / full), nil
}

func yes(v string) bool {
	switch strings.ToLower(v) {
	case "yes", "true":
		return true
	default:
		return false
	}
}

func no(v string) bool {
	switch strings.ToLower(v) {
	case "no", "false":
		return true
	default:
		return false
	}
}
