package experiments

import (
	"context"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Doctor is `lewkit experiments doctor`.
type Doctor struct{}

func (Doctor) Description() string {
	return "probe registered drivers (interface => implementation)"
}

func (*Doctor) Run(ctx context.Context) error {
	return runDemo(ctx, scheduleDoctor)
}

func scheduleDoctor(ctx context.Context) error {
	report := driver.Doctor(ctx)
	for _, iface := range report {
		iface := iface
		taskgroup.Go(ctx, shortIface(iface.Name), taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			return checkIface(ctx, s, iface)
		})
	}
	return nil
}

func checkIface(ctx context.Context, s *taskgroup.Status, iface driver.InterfaceStatus) error {
	n := int64(len(iface.Drivers))
	s.Progress(0, n)
	s.Update("probing")
	selected := ""
	_ = taskgroup.Isolate(ctx, func(ctx context.Context) error {
		for _, d := range iface.Drivers {
			d := d
			taskgroup.Go(ctx, d.ID, taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
				st.Update(d.Name)
				if err := sleep(ctx, 80*time.Millisecond); err != nil {
					return err
				}
				if d.Selected {
					st.Update("selected")
					st.Progress(1, 1)
					return nil
				}
				if d.Available {
					st.Update("available")
					st.Progress(1, 1)
					return nil
				}
				st.Update("skip")
				if d.Error != nil {
					return d.Error
				}
				return driver.ErrIncompatible
			})
			if d.Selected {
				selected = d.ID
			}
		}
		return nil
	})
	s.Progress(n, n)
	if selected != "" {
		s.Update("=> " + selected)
		return nil
	}
	s.Update("=> none")
	return nil
}

func shortIface(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return name
}
