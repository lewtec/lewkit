package pick

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

	assert.True(t, Dir(vendor, true))
	assert.False(t, Dir(goFile, false))
	assert.True(t, File(goFile, false))
	assert.False(t, File(vendor, true))

	assert.True(t, Match("**/*.go")(goFile, false))
	assert.False(t, Match("**/*.go")(txt, false))

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

func TestDerived(t *testing.T) {
	t.Parallel()
	src := path.New("src")
	goFile := path.New("src", "a.go")
	assert.True(t, Glob("**/*.go")(src, true))
	assert.True(t, Glob("**/*.go")(goFile, false))
	assert.Equal(t, Glob("**/*.go")(goFile, false), Or(Dir, Match("**/*.go"))(goFile, false))
	assert.Equal(t, Prune("vendor")(path.New("vendor"), true), Not(And(Dir, Match("vendor")))(path.New("vendor"), true))
}
