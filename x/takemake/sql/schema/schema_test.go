package schema

import (
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/takemake/sql"
	"github.com/stretchr/testify/require"
)

func TestScaffold(t *testing.T) {
	t.Parallel()
	db := New()
	require.NotNil(t, db)
	require.ErrorIs(t, db.Apply(sql.NewFile()), errs.ErrScaffold)
	_, err := db.Make()
	require.ErrorIs(t, err, errs.ErrScaffold)
}
