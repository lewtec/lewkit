package experiments

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWelcomeUsage(t *testing.T) {
	text, err := cmd.Usage[welcomeCmd]("lewkit experiments welcome")
	require.NoError(t, err)
	assert.Contains(t, text, "pick a folder")
}
