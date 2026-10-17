package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitURLWindowsDrive(t *testing.T) {
	scheme, dsn := splitURL(`C:\db\app.decomp`)
	require.Equal(t, "sqlite", scheme)
	require.Equal(t, "C:/db/app.decomp", dsn)

	scheme, dsn = splitURL("C:/db/app.decomp")
	require.Equal(t, "sqlite", scheme)
	require.Equal(t, "C:/db/app.decomp", dsn)

	scheme, dsn = splitURL(`sqlite://C:\db\app.decomp`)
	require.Equal(t, "sqlite", scheme)
	require.Equal(t, "C:/db/app.decomp", dsn)

	scheme, dsn = splitURL("sqlite://C:/db/app.decomp")
	require.Equal(t, "sqlite", scheme)
	require.Equal(t, "C:/db/app.decomp", dsn)

	scheme, dsn = splitURL("postgres://localhost/app")
	require.Equal(t, "postgres", scheme)
	require.Equal(t, "postgres://localhost/app", dsn)

	scheme, dsn = splitURL(":memory:")
	require.Equal(t, "sqlite", scheme)
	require.Equal(t, ":memory:", dsn)
}
