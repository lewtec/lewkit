package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNameDefault(t *testing.T) {
	t.Setenv("LEWKIT_NAME", "")
	previous := name
	name = ""
	t.Cleanup(func() { name = previous })
	require.Equal(t, defaultName, Name())
}

func TestNameEnv(t *testing.T) {
	t.Setenv("LEWKIT_NAME", "myapp")
	previous := name
	name = ""
	t.Cleanup(func() { name = previous })
	require.Equal(t, "myapp", Name())
}

func TestNameStampWins(t *testing.T) {
	t.Setenv("LEWKIT_NAME", "fromenv")
	previous := name
	name = "stamped"
	t.Cleanup(func() { name = previous })
	require.Equal(t, "stamped", Name())
}

func TestNameInvalidFallsBack(t *testing.T) {
	previous := name
	t.Cleanup(func() { name = previous })

	t.Setenv("LEWKIT_NAME", "../x")
	name = ""
	require.Equal(t, defaultName, Name())

	t.Setenv("LEWKIT_NAME", "myapp")
	name = "has-hyphen"
	require.Equal(t, "myapp", Name())
}
