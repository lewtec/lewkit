// Tree schedules a release, then fetch, compile, and package.
//
//	go run ./cmd/lewkit release run --config ./examples/tree/eletrocromo.json
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/examples/internal/pace"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func init() { entry.Bind(run) }

func main() { entry.Main(run) }

func run(ctx context.Context) error {
	const step = 350 * time.Millisecond
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
					if err := pace.Sleep(ctx, step); err != nil {
						return err
					}
				}
				return nil
			})
			taskgroup.Go(ctx, "checksum", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("sha256")
				s.Progress(0, 2)
				if err := pace.Sleep(ctx, step); err != nil {
					return err
				}
				s.Progress(1, 2)
				s.Update("verify")
				if err := pace.Sleep(ctx, step); err != nil {
					return err
				}
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
					if err := pace.Sleep(ctx, step); err != nil {
						return err
					}
				}
				return nil
			})
			types := taskgroup.Go(ctx, "typecheck", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("infer")
				s.Progress(0, 1)
				if err := pace.Sleep(ctx, step); err != nil {
					return err
				}
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
							if err := pace.Sleep(ctx, step); err != nil {
								return struct{}{}, err
							}
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
					if err := pace.Sleep(ctx, step); err != nil {
						return err
					}
				}
				return nil
			})
			taskgroup.Go(ctx, "sign", taskgroup.CPU, func(ctx context.Context, s *taskgroup.Status) error {
				s.Update("minisign")
				done := s.Unit()
				if err := pace.Sleep(ctx, step); err != nil {
					return err
				}
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
