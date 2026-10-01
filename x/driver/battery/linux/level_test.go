package linux

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

func TestParseCapacity(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
		err  error
	}{
		{name: "charge", in: "73\n", want: 73},
		{name: "empty", in: "0\n", want: 0},
		{name: "full", in: "100", want: 100},
		{name: "spaces", in: " 42 \n", want: 42},
		{name: "over", in: "140\n", want: 100},
		{name: "negative", in: "-1\n", err: battery.ErrUnknownLevel},
		{name: "text", in: "full\n", err: strconv.ErrSyntax},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCapacity(tt.in)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPercentOf(t *testing.T) {
	tests := []struct {
		name string
		now  int64
		full int64
		want int
		err  error
	}{
		{name: "half", now: 5000, full: 10000, want: 50},
		{name: "empty", now: 0, full: 10000, want: 0},
		{name: "full", now: 10000, full: 10000, want: 100},
		{name: "over", now: 15000, full: 10000, want: 100},
		{name: "one third", now: 1, full: 3, want: 33},
		{name: "two thirds", now: 2, full: 3, want: 67},
		{name: "negative", now: -1, full: 10, err: battery.ErrUnknownLevel},
		{name: "zero full", now: 1, full: 0, err: battery.ErrUnknownLevel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := percentOf(tt.now, tt.full)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLevelIn(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		_, err := levelIn(t.TempDir())
		require.ErrorIs(t, err, battery.ErrUnknownLevel)
	})
	t.Run("capacity", func(t *testing.T) {
		dir := writeGauges(t, map[string]string{
			"capacity":    "42\n",
			"energy_now":  "1\n",
			"energy_full": "1\n",
		})
		got, err := levelIn(dir)
		require.NoError(t, err)
		require.Equal(t, 42, got)
	})
	t.Run("energy", func(t *testing.T) {
		dir := writeGauges(t, map[string]string{
			"energy_now":  "250\n",
			"energy_full": "1000\n",
		})
		got, err := levelIn(dir)
		require.NoError(t, err)
		require.Equal(t, 25, got)
	})
	t.Run("charge", func(t *testing.T) {
		dir := writeGauges(t, map[string]string{
			"charge_now":  "1\n",
			"charge_full": "2\n",
		})
		got, err := levelIn(dir)
		require.NoError(t, err)
		require.Equal(t, 50, got)
	})
	t.Run("energy now without full uses charge", func(t *testing.T) {
		dir := writeGauges(t, map[string]string{
			"energy_now":  "10\n",
			"charge_now":  "50\n",
			"charge_full": "100\n",
		})
		got, err := levelIn(dir)
		require.NoError(t, err)
		require.Equal(t, 50, got)
	})
	t.Run("bad capacity", func(t *testing.T) {
		dir := writeGauges(t, map[string]string{
			"capacity":    "nope\n",
			"energy_now":  "1\n",
			"energy_full": "1\n",
		})
		_, err := levelIn(dir)
		require.Error(t, err)
		require.NotErrorIs(t, err, os.ErrNotExist)
	})
	t.Run("zero energy full", func(t *testing.T) {
		dir := writeGauges(t, map[string]string{
			"energy_now":  "1\n",
			"energy_full": "0\n",
			"charge_now":  "50\n",
			"charge_full": "100\n",
		})
		_, err := levelIn(dir)
		require.ErrorIs(t, err, battery.ErrUnknownLevel)
	})
}

func writeGauges(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	return dir
}
