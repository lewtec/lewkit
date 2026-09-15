package fs

import (
	"testing"

	"github.com/lewtec/lewkit/x/path"

	"github.com/stretchr/testify/assert"
)

func TestKeepAlgebra(t *testing.T) {
	t.Parallel()
	goFile := path.New("src", "a.go")
	txt := path.New("src", "a.txt")
	vendor := path.New("vendor")
	vendored := path.New("vendor", "a.go")

	k := Glob("**/*.go").And(Prune("vendor"))
	assert.True(t, k(goFile, false))
	assert.False(t, k(txt, false))
	assert.False(t, k(vendor, true))
	assert.True(t, k(path.New("src"), true))

	assert.True(t, pruned(vendored, k))
	assert.False(t, pruned(goFile, k))

	either := Glob("**/*.go").Or(Glob("**/*.txt"))
	assert.True(t, either(goFile, false))
	assert.True(t, either(txt, false))

	notGo := Glob("**/*.go").Not()
	assert.False(t, notGo(goFile, false))
	assert.True(t, notGo(txt, false))
}
