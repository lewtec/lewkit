package native

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type toy struct {
	drops int
}

func (t *toy) Drop() { t.drops++ }

func TestChildrenAddDropsOnce(t *testing.T) {
	var set Children
	a := &toy{}
	set.Add(a)
	set.Add(a)
	set.Drop()
	require.Equal(t, 1, a.drops)
	set.Drop()
	require.Equal(t, 1, a.drops)
}

func TestChildrenForgetSkipsDrop(t *testing.T) {
	var set Children
	a := &toy{}
	b := &toy{}
	set.Add(a)
	set.Add(b)
	set.Forget(a)
	set.Drop()
	require.Equal(t, 0, a.drops)
	require.Equal(t, 1, b.drops)
}

func TestChildrenIgnoreNil(t *testing.T) {
	var set Children
	set.Add(nil)
	var none *toy
	set.Add(none)
	set.Forget(nil)
	set.Drop()
}
