package gui

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/stretchr/testify/require"
)

func TestCounterFollowsScheme(t *testing.T) {
	counter, err := NewCounter()
	require.NoError(t, err)
	dark := counter.View().(*Box)
	require.Equal(t, Color{28, 28, 34, 255}, *dark.Fill)
	next, cmd := counter.Update(SchemeMsg{Scheme: appearance.Light})
	require.Nil(t, cmd)
	light := next.View().(*Box)
	require.Equal(t, Color{246, 246, 244, 255}, *light.Fill)
	text := light.Child.(*Flex).Children[2].Child.(*Box).Child.(*Text)
	require.Equal(t, Color{28, 28, 34, 255}, text.Ink)
}
