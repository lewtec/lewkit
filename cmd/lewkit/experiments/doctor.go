package experiments

import (
	"context"
	"os"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

// Doctor is `lewkit experiments doctor`.
type Doctor struct{}

func (Doctor) Description() string {
	return "list drivers (interface => implementation)"
}

func (*Doctor) Run(ctx context.Context) error {
	_, err := os.Stdout.WriteString(progress.Format(doctorNodes(driver.Doctor(ctx)), 0))
	return err
}

func doctorNodes(report []driver.InterfaceStatus) []taskgroup.Node {
	var nodes []taskgroup.Node
	var id taskgroup.ID
	next := func() taskgroup.ID {
		id++
		return id
	}
	for _, iface := range report {
		parent := next()
		selected := ""
		ok := 0
		for _, d := range iface.Drivers {
			if d.Selected {
				selected = d.ID
			}
			if d.Available {
				ok++
			}
		}
		msg := "=> none"
		st := taskgroup.Failed
		if selected != "" {
			msg = "=> " + selected
			st = taskgroup.Done
		}
		nodes = append(nodes, taskgroup.Node{
			ID:           parent,
			Name:         shortIface(iface.Name),
			Message:      msg,
			State:        st,
			Pool:         taskgroup.Control,
			Current:      int64(ok),
			Total:        int64(len(iface.Drivers)),
			LiveChildren: len(iface.Drivers),
		})
		for _, d := range iface.Drivers {
			child := taskgroup.Node{
				ID:     next(),
				Parent: parent,
				Name:   d.ID,
				Pool:   taskgroup.CPU,
			}
			switch {
			case d.Selected:
				child.Message = "selected"
				child.State = taskgroup.Done
			case d.Available:
				child.Message = "available"
				child.State = taskgroup.Done
			default:
				child.Message = "skip"
				if d.Error != nil {
					child.Message = d.Error.Error()
				}
				child.State = taskgroup.Failed
			}
			nodes = append(nodes, child)
		}
	}
	return nodes
}

func shortIface(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}
