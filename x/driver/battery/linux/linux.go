package linux

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/battery"
)

type backend struct{}

func batteryDir() (string, error) {
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*/status")
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", battery.ErrNoBattery
	}
	return filepath.Dir(matches[0]), nil
}

func (backend) BatteryStatus(context.Context) (battery.Status, error) {
	dir, err := batteryDir()
	if err != nil {
		return battery.Unknown, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "status"))
	if err != nil {
		return battery.Unknown, err
	}
	return parseStatus(string(data)), nil
}

func (backend) BatteryLevel(context.Context) (int, error) {
	dir, err := batteryDir()
	if err != nil {
		return 0, err
	}
	return levelIn(dir)
}

func parseStatus(text string) battery.Status {
	switch strings.TrimSpace(text) {
	case "Charging":
		return battery.Charging
	case "Discharging":
		return battery.Discharging
	case "Full":
		return battery.Full
	default:
		return battery.Status(strings.TrimSpace(text))
	}
}

// Missing files fall through. Any other error stops the search.
func levelIn(dir string) (int, error) {
	n, err := readCapacity(filepath.Join(dir, "capacity"))
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return n, err
	}
	n, err = readPair(dir, "energy_now", "energy_full")
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return n, err
	}
	n, err = readPair(dir, "charge_now", "charge_full")
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return n, err
	}
	return 0, battery.ErrUnknownLevel
}

func readCapacity(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseCapacity(string(data))
}

func parseCapacity(text string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("capacity: %w", err)
	}
	if n < 0 {
		return 0, battery.ErrUnknownLevel
	}
	if n > 100 {
		return 100, nil
	}
	return n, nil
}

func readPair(dir, nowName, fullName string) (int, error) {
	now, err := readInt(filepath.Join(dir, nowName))
	if err != nil {
		return 0, err
	}
	full, err := readInt(filepath.Join(dir, fullName))
	if err != nil {
		return 0, err
	}
	return percentOf(now, full)
}

func readInt(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return n, nil
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
