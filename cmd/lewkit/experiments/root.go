package experiments

import "github.com/lewtec/lewkit/x/taskgroup"

// Command is `lewkit experiments`.
type Command struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
	Demo          *Demo
	Window        *Window
	Compute       *Compute
}

func (Command) Description() string {
	return "experimental features and prototypes"
}
