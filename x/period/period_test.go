package period

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func d(s string) time.Time {
	t, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseDate(t *testing.T) {
	got, err := ParseDate("")
	require.NoError(t, err)
	require.True(t, got.IsZero())
	got, err = ParseDate("2024-07-15")
	require.NoError(t, err)
	want := time.Date(2024, 7, 15, 0, 0, 0, 0, time.UTC)
	require.Equal(t, want, got)
	_, err = ParseDate("not-a-date")
	require.Error(t, err)
}

func TestParseYearMonth(t *testing.T) {
	now := d("2024-07-15")
	r, err := Parse("2024", now)
	require.NoError(t, err)
	require.Equal(t, d("2024-01-01"), r.Start)
	require.Equal(t, d("2024-12-31"), r.End)
	r, err = Parse("2024-03", now)
	require.NoError(t, err)
	require.Equal(t, d("2024-03-01"), r.Start)
	require.Equal(t, d("2024-03-31"), r.End)
	r, err = Parse("2024-Q2", now)
	require.NoError(t, err)
	require.Equal(t, d("2024-04-01"), r.Start)
	require.Equal(t, d("2024-06-30"), r.End)
}

func TestParseRelativeAndRange(t *testing.T) {
	now := d("2024-07-15")
	r, err := Parse("month", now)
	require.NoError(t, err)
	require.Equal(t, d("2024-07-01"), r.Start)
	require.Equal(t, d("2024-07-31"), r.End)
	r, err = Parse("month-1", now)
	require.NoError(t, err)
	require.Equal(t, d("2024-06-01"), r.Start)
	require.Equal(t, d("2024-06-30"), r.End)
	r, err = Parse("year-1", now)
	require.NoError(t, err)
	require.Equal(t, d("2023-01-01"), r.Start)
	require.Equal(t, d("2023-12-31"), r.End)
	r, err = Parse("2023 - 2024-03", now)
	require.NoError(t, err)
	require.Equal(t, d("2023-01-01"), r.Start)
	require.Equal(t, d("2024-03-31"), r.End)
}

func TestParseDayWeek(t *testing.T) {
	now := d("2024-07-15")
	r, err := Parse("2024-07-15", now)
	require.NoError(t, err)
	require.Equal(t, d("2024-07-15"), r.Start)
	require.Equal(t, d("2024-07-15"), r.End)
	r, err = Parse("2024-W29", now)
	require.NoError(t, err)
	// ISO week 29 of 2024 starts Monday 2024-07-15
	require.Equal(t, d("2024-07-15"), r.Start)
	require.Equal(t, d("2024-07-21"), r.End)
}
