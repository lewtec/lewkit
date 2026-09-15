package io

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMkdirp(t *testing.T) {
	tempdir := t.TempDir()
	assert.NoError(t, Mkdirp(tempdir))
	testdir := path.Join(tempdir, "picuinha")
	assert.NoError(t, Mkdirp(testdir))
	assert.DirExists(t, testdir)
}

func TestMkdirpParents(t *testing.T) {
	nested := path.Join(t.TempDir(), "a", "b", "c")
	assert.NoError(t, Mkdirp(nested))
	assert.DirExists(t, nested)
	assert.NoError(t, Mkdirp(nested))
}
