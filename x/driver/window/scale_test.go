package window

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaleMath(t *testing.T) {
	assert.Equal(t, 1.0, Scale(0).Factor())
	assert.Equal(t, 1.0, Scale(-3).Factor())
	assert.Equal(t, 1.0, Scale(math.NaN()).Factor())
	assert.Equal(t, 1.0, Scale(math.Inf(1)).Factor())
	assert.Equal(t, 2.0, Scale(2).Factor())

	assert.Equal(t, 0, Scale(2).Device(0))
	assert.Equal(t, 1, Scale(0.1).Device(1))
	assert.Equal(t, 16, Scale(2).Device(8))
	assert.Equal(t, 12, Scale(2).Device(6))
	assert.Equal(t, 3, Scale(1.5).Device(2))
	assert.Equal(t, 8, Scale(2).Logical(16))
	assert.Equal(t, 1, Scale(3).Logical(1))
	assert.Equal(t, -2, Scale(2).Device(-1))
}

func TestSetScaleEmitsDensity(t *testing.T) {
	buf := NewBuffer(8, 6)
	assert.Equal(t, 1.0, buf.Scale().Factor())

	ctx := t.Context()
	events := buf.Subscribe(ctx)
	buf.SetScale(1)
	require.Equal(t, Density{Scale: 1}, readEvent(t, events))
	assert.Equal(t, 1.0, buf.Scale().Factor())

	buf.SetScale(1)
	assertNoEvent(t, events)

	buf.SetScale(2)
	require.Equal(t, Density{Scale: 2}, readEvent(t, events))
	assert.Equal(t, Scale(2), buf.Scale())

	require.NoError(t, buf.Close())
	drainClose(t, events)
	buf.SetScale(3)
	assert.Equal(t, Scale(2), buf.Scale())
	assertNoEvent(t, events)
}

func TestSetFontScaleDoesNotChangeDensity(t *testing.T) {
	buf := NewBuffer(8, 6)
	assert.Equal(t, 1.0, buf.FontScale().Factor())

	ctx := t.Context()
	events := buf.Subscribe(ctx)
	buf.SetScale(2)
	require.Equal(t, Density{Scale: 2}, readEvent(t, events))
	buf.SetFontScale(1.3)
	require.Equal(t, FontScale{Scale: Scale(1.3)}, readEvent(t, events))
	assert.Equal(t, Scale(2), buf.Scale())
	assert.Equal(t, Scale(1.3), buf.FontScale())

	buf.SetFontScale(1.3)
	assertNoEvent(t, events)

	buf.SetFontScale(0)
	require.Equal(t, FontScale{Scale: 1}, readEvent(t, events))
	assert.Equal(t, 1.0, buf.FontScale().Factor())

	require.NoError(t, buf.Close())
	drainClose(t, events)
	buf.SetFontScale(2)
	assert.Equal(t, 1.0, buf.FontScale().Factor())
	assertNoEvent(t, events)
}

func readEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()
	select {
	case ev := <-events:
		return ev
	case <-time.After(time.Second):
		t.Fatal("missing event")
		return nil
	}
}

func drainClose(t *testing.T, events <-chan Event) {
	t.Helper()
	ev := readEvent(t, events)
	require.Equal(t, Close{}, ev)
}

func assertNoEvent(t *testing.T, events <-chan Event) {
	t.Helper()
	select {
	case ev := <-events:
		t.Fatalf("unexpected event %T", ev)
	default:
	}
}
