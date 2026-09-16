package experiments

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/taskgroup"
)

type treeCmd struct{}

func (treeCmd) Description() string {
	return "deep nested tree: release → fetch/compile/package"
}

func (*treeCmd) Run(ctx context.Context) error {
	return runDemo(ctx, scheduleTree)
}

const treeStep = 350 * time.Millisecond

func scheduleTree(ctx context.Context) error {
	taskgroup.Go(ctx, "release", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
		s.Update("orchestrating")
		s.Progress(0, 3)

		fetch := taskgroup.Go(ctx, "fetch", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("mirrors")
			s.Progress(0, 2)
			reg := taskgroup.Go(ctx, "registry", taskgroup.Internet, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("GET /mods")
				s.Progress(0, 4)
				for i := 1; i <= 4; i++ {
					s.Progress(int64(i), 4)
					s.Update(fmt.Sprintf("page %d/4", i))
					time.Sleep(treeStep)
				}
				return nil
			})
			taskgroup.Go(ctx, "checksum", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("sha256")
				s.Progress(0, 2)
				time.Sleep(treeStep)
				s.Progress(1, 2)
				s.Update("verify")
				time.Sleep(treeStep)
				s.Progress(2, 2)
				return nil
			}, reg)
			s.Progress(1, 2)
			s.Update("waiting checksum")
			return nil
		})

		compile := taskgroup.Go(ctx, "compile", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("frontend")
			s.Progress(0, 3)
			parse := taskgroup.Go(ctx, "parse", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("syntax")
				s.Progress(0, 3)
				for i := 1; i <= 3; i++ {
					s.Progress(int64(i), 3)
					time.Sleep(treeStep)
				}
				return nil
			})
			types := taskgroup.Go(ctx, "typecheck", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("infer")
				s.Progress(0, 1)
				time.Sleep(treeStep)
				s.Progress(1, 1)
				return nil
			}, parse)
			taskgroup.Go(ctx, "codegen", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("targets")
				s.Progress(0, 2)
				_, err := taskgroup.Map[string, struct{}]{
					Name:     "isa",
					Items:    []string{"amd64", "arm64"},
					PoolKind: taskgroup.CPU,
					TaskName: func(_ int, isa string) string { return isa },
					Fn: func(ctx context.Context, st *taskgroup.Status, isa string) (struct{}, error) {
						st.Update("emit " + isa)
						st.Progress(0, 3)
						for i := 1; i <= 3; i++ {
							st.Progress(int64(i), 3)
							time.Sleep(treeStep)
						}
						return struct{}{}, nil
					},
				}.Run(ctx)
				if err != nil {
					return err
				}
				s.Progress(2, 2)
				return nil
			}, types)
			s.Progress(1, 3)
			return nil
		}, fetch)

		taskgroup.Go(ctx, "package", taskgroup.Control, func(ctx context.Context, s *taskgroup.Status) error {
			s.Update("artifacts")
			s.Progress(0, 2)
			tar := taskgroup.Go(ctx, "tarball", taskgroup.IO, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("tar czf")
				s.Progress(0, 5)
				for i := 1; i <= 5; i++ {
					s.Progress(int64(i), 5)
					time.Sleep(treeStep)
				}
				return nil
			})
			taskgroup.Go(ctx, "sign", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("minisign")
				done := s.Unit()
				time.Sleep(treeStep)
				done()
				return nil
			}, tar)
			s.Progress(1, 2)
			return nil
		}, compile)

		s.Progress(3, 3)
		s.Update("waiting children")
		return nil
	})
	return nil
}
