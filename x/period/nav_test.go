package period

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShiftMonth(t *testing.T) {
	now := d("2024-07-15")
	got, err := Shift("2024-03", now, 1)
	require.NoError(t, err)
	require.Equal(t, "2024-04", got)
	got, err = Shift("2024-03", now, -1)
	require.NoError(t, err)
	require.Equal(t, "2024-02", got)
	got, err = Shift("", now, 0)
	require.NoError(t, err)
	// empty + shift 0 via At month of now after defaulting — Shift with delta 0 on empty starts at current month
	// Shift empty with delta 0: sets to current month then shifts 0
	require.Equal(t, "2024-07", got)
}

func TestSetInterval(t *testing.T) {
	now := d("2024-07-15")
	require.Equal(t, "2024", SetInterval("2024-03", now, KindYear))
	require.Equal(t, "2024-Q1", SetInterval("2024-03", now, KindQuarter))
	require.Equal(t, "2024-07", SetInterval("", now, KindMonth))
}

func TestDisplayLabel(t *testing.T) {
	now := d("2024-07-15")
	require.Equal(t, "March 2024", DisplayLabel("2024-03", now))
	require.Equal(t, "All time", DisplayLabel("", now))
}

func TestKind(t *testing.T) {
	now := time.Now()
	require.Equal(t, KindMonth, Classify("2024-03", now))
	require.Equal(t, KindMonth, Classify("month-1", now))
}
