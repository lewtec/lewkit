package taskgroup

import (
	"cmp"
	"context"
	"strconv"
	"sync/atomic"
)

// Map fans out Items under one Control node and collects results.
// Children attach as tree children of that node, so List(n) shows the
// parent and as many live items as still fit.
//
// Only leaves take IO/CPU/Internet. If Fn schedules more limited-pool
// work, PoolKind must be Control.
type Map[T any, U any] struct {
	Name     string
	Items    []T
	Pool     func(T) PoolKind
	PoolKind PoolKind
	Serial   bool
	TaskName func(int, T) string
	Fn       func(ctx context.Context, s *Status, item T) (U, error)
}

func (m Map[T, U]) poolFor(item T) PoolKind {
	if m.Pool != nil {
		return m.Pool(item)
	}
	return m.PoolKind
}

func (m Map[T, U]) childName(i int, item T, name string) string {
	if m.TaskName != nil {
		if n := m.TaskName(i, item); n != "" {
			return n
		}
	}
	return name + ":" + strconv.Itoa(i)
}

// Run schedules the map on MustFromContext(ctx) and blocks until complete.
// An empty Items list returns a non-nil empty slice.
func (m Map[T, U]) Run(ctx context.Context) ([]U, error) {
	if m.Fn == nil {
		return nil, ErrNilFn
	}
	if len(m.Items) == 0 {
		return []U{}, nil
	}

	name := cmp.Or(m.Name, "map")
	total := int64(len(m.Items))
	results := make([]U, len(m.Items))

	id := Go(ctx, name, Control, func(ctx context.Context, s *Status) error {
		s.Update("0/" + strconv.FormatInt(total, 10))
		s.Progress(0, total)

		var completed atomic.Int64
		var prev ID
		for i, item := range m.Items {
			var deps []ID
			if m.Serial && prev != 0 {
				deps = []ID{prev}
			}
			child := Go(ctx, m.childName(i, item, name), m.poolFor(item), func(ctx context.Context, itemStatus *Status) error {
				u, err := m.Fn(ctx, itemStatus, item)
				if err != nil {
					return err
				}
				results[i] = u
				cur := completed.Add(1)
				s.Progress(cur, total)
				s.Update(strconv.FormatInt(cur, 10) + "/" + strconv.FormatInt(total, 10))
				return nil
			}, deps...)
			if m.Serial {
				prev = child
			}
		}

		err := MustFromContext(ctx).waitLive(ctx, taskFromContext(ctx))
		if err == nil {
			s.Progress(total, total)
			s.Update(strconv.FormatInt(total, 10) + "/" + strconv.FormatInt(total, 10))
		}
		return err
	})
	if err := MustFromContext(ctx).waitTask(ctx, id); err != nil {
		return nil, err
	}
	return results, nil
}

// Each fans out Items under one Control node without collecting results.
type Each[T any] struct {
	Name     string
	Items    []T
	Pool     func(T) PoolKind
	PoolKind PoolKind
	Serial   bool
	TaskName func(int, T) string
	Fn       func(ctx context.Context, s *Status, item T) error
}

// Run schedules the work on MustFromContext(ctx) and blocks until complete.
func (e Each[T]) Run(ctx context.Context) error {
	if e.Fn == nil {
		return ErrNilFn
	}
	_, err := Map[T, struct{}]{
		Name:     e.Name,
		Items:    e.Items,
		Pool:     e.Pool,
		PoolKind: e.PoolKind,
		Serial:   e.Serial,
		TaskName: e.TaskName,
		Fn: func(ctx context.Context, s *Status, item T) (struct{}, error) {
			return struct{}{}, e.Fn(ctx, s, item)
		},
	}.Run(ctx)
	return err
}
