package experiments

import (
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinStep(t *testing.T) {
	model, err := newSpinModel(triangleTurnsPerSecond, 32, 32, triangleDynamic)
	require.NoError(t, err)
	model.bump(triangleTurnStep)
	assert.InDelta(t, 0.30, model.scale, 1e-9)
	model.bump(-triangleTurnStep)
	model.bump(-triangleTurnStep)
	assert.InDelta(t, 0.20, model.scale, 1e-9)
}

func TestSpinIntegratesRate(t *testing.T) {
	model, err := newSpinModel(1, 8, 8, triangleDynamic)
	require.NoError(t, err)
	_, _ = model.Update(gui.TickMsg{Elapsed: time.Second, Size: model.size, Period: time.Second / 60})
	assert.InDelta(t, 1, model.turn, 1e-6)
	model.scale = 0.5
	_, _ = model.Update(gui.TickMsg{Elapsed: 2 * time.Second, Size: model.size, Period: time.Second / 60})
	assert.InDelta(t, 1.5, model.turn, 1e-6)
}
