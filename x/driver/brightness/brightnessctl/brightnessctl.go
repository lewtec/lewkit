package brightnessctl

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/brightness"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

type backend struct{}

func (backend) Status(ctx context.Context) (*brightness.Device, error) {
	out, err := execdriver.OutputString(ctx, "brightnessctl", "-m")
	if err != nil {
		return nil, fmt.Errorf("get brightness status: %w", err)
	}
	return parseStatus(out)
}

func (backend) SetBrightness(ctx context.Context, level float64) error {
	percent := fmt.Sprintf("%d%%", int(level*100))
	if err := execdriver.RunProgram(ctx, "brightnessctl", "s", percent); err != nil {
		return fmt.Errorf("set brightness: %w", err)
	}
	return nil
}

func parseStatus(out string) (*brightness.Device, error) {
	text := strings.TrimSpace(out)
	if text == "" {
		return nil, brightness.ErrDeviceNotFound
	}
	for _, line := range strings.Split(text, "\n") {
		parts := strings.Split(line, ",")
		if len(parts) < 5 {
			continue
		}
		level, err := strconv.Atoi(strings.TrimSuffix(parts[3], "%"))
		if err != nil {
			continue
		}
		return &brightness.Device{Name: parts[0], Brightness: float64(level) / 100}, nil
	}
	return nil, brightness.ErrDeviceNotFound
}
