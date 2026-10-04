package launcher

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
)

// ErrCanceled means the user dismissed a prompt or a list.
var ErrCanceled = errors.New("canceled")

// ChoiceLines is one label per line. A newline inside a label becomes a space.
func ChoiceLines(items []Item) string {
	var b strings.Builder
	for i, item := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.ReplaceAll(ItemLabel(item), "\n", " "))
	}
	return b.String()
}

// ConfirmAnswer reads a confirm reply. No and cancel are false.
func ConfirmAnswer(raw string) (bool, error) {
	status, _ := askwire.Split(raw)
	switch status {
	case askwire.StatusYes:
		return true, nil
	case askwire.StatusNo, askwire.StatusCanceled:
		return false, nil
	case askwire.StatusNoActivity, askwire.StatusBusy:
		return false, fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	default:
		return false, fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	}
}

// PromptAnswer reads a prompt reply.
func PromptAnswer(raw string) (string, error) {
	status, payload := askwire.Split(raw)
	switch status {
	case askwire.StatusOK:
		return payload, nil
	case askwire.StatusCanceled:
		return "", ErrCanceled
	case askwire.StatusNoActivity, askwire.StatusBusy:
		return "", fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	default:
		return "", fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	}
}

// ChooseAnswer reads a list reply. Cancel is a nil item.
func ChooseAnswer(items []Item, raw string) (*Item, error) {
	status, payload := askwire.Split(raw)
	switch status {
	case askwire.StatusCanceled:
		return nil, nil
	case askwire.StatusOK:
		return MatchSelected(items, payload), nil
	case askwire.StatusNoActivity, askwire.StatusBusy:
		return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	default:
		return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	}
}
