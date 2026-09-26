package table

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type row struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	When time.Time
}

type rowSpec struct {
	When Field[time.Time]
	ID   Field[string]
	Name Field[string]
}

func TestMakeBuildsViewOnce(t *testing.T) {
	when := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	view := Must(Make[row](rowSpec{
		When: Field[time.Time]{Format: time.DateOnly},
		Name: Field[string]{Format: "%q"},
	}))
	seq := func(yield func(row) bool) {
		yield(row{ID: "z", Name: "a", When: when})
	}
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, JSONL, seq, view))
	assert.Equal(t, "{\"When\":\"2026-09-26\",\"id\":\"z\",\"name\":\"\\\"a\\\"\"}\n", buf.String())
	assert.Equal(t, []string{"When", "id", "name"}, namesOf(view))
}

func TestMakeRejectsMismatch(t *testing.T) {
	type badSpec struct {
		ID Field[int]
	}
	_, err := Make[row](badSpec{})
	assert.ErrorIs(t, err, ErrColumn)
}

func TestMakeRejectsMissingField(t *testing.T) {
	type badSpec struct {
		Missing Field[string]
	}
	_, err := Make[row](badSpec{})
	assert.ErrorIs(t, err, ErrColumn)
}

func namesOf(view View[row]) []string {
	cols := view.Columns()
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = col.Name
	}
	return out
}
