package notify_send

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/lewtec/lewkit/x/driver/notification"
)

type backend struct{}

func (backend) Notify(ctx context.Context, n notification.Notification) error {
	args := []string{}
	if n.Urgency != "" {
		args = append(args, "-u", n.Urgency)
	}
	if n.Icon != "" {
		args = append(args, "-i", n.Icon)
	}
	if n.HasProgress {
		args = append(args, "-h", fmt.Sprintf("int:value:%d", int(n.Progress*100)))
	}
	if n.ID != 0 {
		args = append(args, "-r", strconv.FormatUint(uint64(n.ID), 10))
	}
	args = append(args, n.Title, n.Message)
	cmd := exec.CommandContext(ctx, "notify-send", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("notify-send: %w", err)
	}
	return nil
}
