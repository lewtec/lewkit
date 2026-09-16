package ndfa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphNil(t *testing.T) {
	var automaton *NDFA
	assert.Empty(t, automaton.Graph().Nodes)
}
