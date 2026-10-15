package period

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func day(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		panic(err)
	}
	return t
}

func TestChartBinFromFilter(t *testing.T) {
	cases := []struct {
		raw  string
		r    Range
		want BinKind
	}{
		{"2024", Range{Start: day("2024-01-01"), End: day("2024-12-31")}, BinMonth},
		{"2024-03", Range{Start: day("2024-03-01"), End: day("2024-03-31")}, BinDay},
		{"2024-Q1", Range{Start: day("2024-01-01"), End: day("2024-03-31")}, BinMonth},
		{"2020 - 2024", Range{Start: day("2020-01-01"), End: day("2024-12-31")}, BinYear},
		{"", Range{}, BinMonth},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, ChartBin(c.raw, c.r))
	}
}

func TestIterateBinsMonth(t *testing.T) {
	r := Range{Start: day("2024-01-15"), End: day("2024-03-10")}
	bins := IterateBins(r, BinMonth)
	require.Len(t, bins, 3)
	require.Equal(t, day("2024-01-15"), bins[0].Start)
	require.Equal(t, day("2024-01-31"), bins[0].End)
	require.Equal(t, day("2024-03-10"), bins[2].End)
}
