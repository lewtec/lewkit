package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFPSGet(t *testing.T) {
	var fps FPS
	require.Zero(t, fps.Get())
	fps.last = time.Now().Add(-100 * time.Millisecond)
	got := fps.Get()
	require.GreaterOrEqual(t, got, float64(5))
	require.LessOrEqual(t, got, float64(20))
}
