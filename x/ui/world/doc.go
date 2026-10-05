// Package world is one frame of data and the systems that advance it.
//
// This is the part of Bevy's model that fits a lewkit primitive. Bevy's
// App is a world, a schedule, and plugins. Here that root is a [Sim].
// gui.Model stays Init, Update, View. A model may call [Sim.Frame] from
// Update and read columns from View. This package does not import gui,
// tui, web, x/app, or x/driver, and it does not open a window.
//
// A Bevy component is a [Column]: the values of one Go type, addressed
// by [Entity]. SPEC.md already uses component for a bubbletea type, a
// templ template, or a tensor transformer, so this package does not.
// A Bevy resource is [Put] and [Mut]: one value of a Go type on that
// world. That is not x/singleton, which is process-wide.
// A Bevy message is [Send] and [Messages]: values a later system in
// this frame can read. That is not gui.Msg, and it is not x/event.Bus.
// The bus fans out and drops when a subscriber is full. A message here
// stays until the next frame.
//
// [World.Despawn] and [Table.Remove] during a system apply after that
// system returns, so a column walk stays valid. [World.Spawn] and
// [Table.Insert] happen immediately. A walk does not visit rows inserted
// during that walk.
//
// [Query] walks one column. [With] and [Without] keep or drop rows that
// also store another column. [Spawn] and [Spawn2] store columns on a
// new entity. [Init] stores a resource once, so two plugins can declare
// it. [Sim.Step] writes [Time] from a duration the host already measured.
//
// Systems in one [Set] run on the caller goroutine. With no [Reg.After]
// or [Reg.Before], registration order is the order. [Sim.Chain] runs its
// systems in the order written. A later plugin can place a system
// between those functions. A [World] is not safe for concurrent systems.
// Bevy also runs non-overlapping systems in parallel. That executor is
// not this package. x/taskgroup remains the pool for work that outlives
// a frame.
//
// Bevy keeps a message for two frames and gives each reader a cursor, so
// a system scheduled before the sender still observes it. Here the
// sender runs in an earlier set than the reader. The next [Sim.Frame]
// clears the log.
//
// Not in this package: archetype storage, observers, relationship edges,
// states, a fixed timestep, assets, scenes, and a render world. The
// view is the extract, and it lives in the caller.
//
//	func (m *screen) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
//	    if _, ok := msg.(gui.TickMsg); ok {
//	        m.sim.Frame(m.ctx)
//	    }
//	    return m, nil
//	}
package world
