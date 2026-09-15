package keep

import (
	"testing"

	"github.com/lewtec/lewkit/x/path"

	"github.com/stretchr/testify/assert"
)

func TestAlgebra(t *testing.T) {
	t.Parallel()
	goFile := path.New("src", "a.go")
	txt := path.New("src", "a.txt")
	vendor := path.New("vendor")

	k := And(Glob("**/*.go"), Prune("vendor"))
	assert.True(t, k(goFile, false))
	assert.False(t, k(txt, false))
	assert.False(t, k(vendor, true))
	assert.True(t, k(path.New("src"), true))

	either := Or(Glob("**/*.go"), Glob("**/*.txt"))
	assert.True(t, either(goFile, false))
	assert.True(t, either(txt, false))

	notGo := Not(Glob("**/*.go"))
	assert.False(t, notGo(goFile, false))
	assert.True(t, notGo(txt, false))

	assert.True(t, And()(txt, false))
	assert.False(t, Or()(txt, false))
}
