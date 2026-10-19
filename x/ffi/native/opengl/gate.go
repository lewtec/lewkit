//go:build !darwin && !ios

package opengl

import (
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	glNameBuffer = iota
	glNameProgram
)

// glGate is the device handle passed to a cleanup.
// It does not point at *Device. Kids are GL names, not Go buffers,
// so the cleanup argument does not keep the device alive.
type glGate struct {
	mu     sync.Mutex
	closed bool
	run    *runner
	gl     *glAPI
	kids   native.Children
}

// shutdown deletes names on the context thread, then stops that thread.
// A second call does nothing. runner.Stop closes the job channel.
func (g *glGate) shutdown() {
	if g == nil {
		return
	}
	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return
	}
	g.closed = true
	run := g.run
	g.mu.Unlock()
	if run == nil {
		return
	}
	_ = run.Do(func() error {
		g.kids.Drop()
		return nil
	})
	run.Stop()
}

// glName is one buffer or program name. Drop deletes it on the caller thread.
type glName struct {
	mu   sync.Mutex
	done bool
	gate *glGate
	kind int
	id   uint32
}

// Drop deletes the GL name. The caller is on the context thread.
func (n *glName) Drop() {
	if n == nil {
		return
	}
	n.mu.Lock()
	if n.done {
		n.mu.Unlock()
		return
	}
	n.done = true
	gate, kind, id := n.gate, n.kind, n.id
	n.mu.Unlock()
	if gate == nil || gate.gl == nil || id == 0 {
		return
	}
	switch kind {
	case glNameBuffer:
		if gate.gl.deleteBuffers != nil {
			gate.gl.deleteBuffers(1, &id)
		}
	case glNameProgram:
		if gate.gl.deleteProgram != nil {
			gate.gl.deleteProgram(id)
		}
	}
}

func (d *Device) trackGL(kind int, id uint32) *glName {
	if d == nil || d.gate == nil || id == 0 {
		return nil
	}
	name := &glName{gate: d.gate, kind: kind, id: id}
	d.gate.kids.Add(name)
	return name
}
