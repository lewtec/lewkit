package android

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDataDirFor(t *testing.T) {
	const pkg = "br.tec.lew.host"
	user := "/data/user/0/" + pkg
	legacy := "/data/data/" + pkg

	got, err := dataDirFor(pkg, 10123, func(path string) bool { return path == user })
	require.NoError(t, err)
	require.Equal(t, user, got)

	got, err = dataDirFor(pkg, 10123, func(path string) bool { return path == legacy })
	require.NoError(t, err)
	require.Equal(t, legacy, got)

	profile := "/data/user/10/" + pkg
	got, err = dataDirFor(pkg, 10*aidUserOffset+10123, func(path string) bool { return path == profile })
	require.NoError(t, err)
	require.Equal(t, profile, got)

	_, err = dataDirFor(pkg, 10123, func(string) bool { return false })
	require.ErrorIs(t, err, errNoDataDir)

	_, err = dataDirFor("", 10123, func(string) bool { return true })
	require.ErrorIs(t, err, errNoDataDir)
}
