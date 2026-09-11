package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductPositional(t *testing.T) {
	type args struct {
		pair KV[string, *StringArg]
	}
	got, err := Parse[args]("name", "lucas")
	require.NoError(t, err)
	assert.Equal(t, got.pair.K.Value(), "name")
	assert.Equal(t, got.pair.V.Value(), "lucas")
}

func TestProductThenRest(t *testing.T) {
	type args struct {
		pair KV[string, *StringArg]
		rest []StringArg
	}
	got, err := Parse[args]("name", "lucas", "x")
	require.NoError(t, err)
	assert.Equal(t, got.pair.K.Value(), "name")
	assert.Equal(t, got.pair.V.Value(), "lucas")
	assert.Equal(t, Values(got.rest), []string{"x"})
}

func TestSeqKV(t *testing.T) {
	type args struct {
		pairs Seq[KV[string, *StringArg]]
	}
	got, err := Parse[args]("name", "lucas", "age", "26")
	require.NoError(t, err)
	assert.Equal(t, Map(got.pairs), map[string]string{"name": "lucas", "age": "26"})
}

func TestSliceKVSameAsSeq(t *testing.T) {
	type args struct {
		pairs []KV[string, *StringArg]
	}
	got, err := Parse[args]("name", "lucas", "age", "26")
	require.NoError(t, err)
	assert.Equal(t, Map(got.pairs), map[string]string{"name": "lucas", "age": "26"})
}

func TestMapEmpty(t *testing.T) {
	assert.Equal(t, Map([]KV[string, *StringArg](nil)), map[string]string{})
}

func TestSeqIntProduct(t *testing.T) {
	type args struct {
		pairs Seq[KV[int, *IntArg[int]]]
	}
	got, err := Parse[args]("lucas", "26", "ada", "36")
	require.NoError(t, err)
	assert.Equal(t, Map(got.pairs), map[string]int{"lucas": 26, "ada": 36})
}

func TestExactArray(t *testing.T) {
	type args struct {
		pair [2]StringArg
	}
	got, err := Parse[args]("a", "b")
	require.NoError(t, err)
	assert.Equal(t, got.pair[0].Value(), "a")
	assert.Equal(t, got.pair[1].Value(), "b")
}

func TestExactArrayThenRest(t *testing.T) {
	type args struct {
		pair [2]StringArg
		rest []StringArg
	}
	got, err := Parse[args]("a", "b", "c")
	require.NoError(t, err)
	assert.Equal(t, got.pair[0].Value(), "a")
	assert.Equal(t, got.pair[1].Value(), "b")
	assert.Equal(t, Values(got.rest), []string{"c"})
}

func TestExactTwoKV(t *testing.T) {
	type args struct {
		pairs [2]KV[string, *StringArg]
	}
	got, err := Parse[args]("name", "lucas", "age", "26")
	require.NoError(t, err)
	assert.Equal(t, got.pairs[0].K.Value(), "name")
	assert.Equal(t, got.pairs[0].V.Value(), "lucas")
	assert.Equal(t, got.pairs[1].K.Value(), "age")
	assert.Equal(t, got.pairs[1].V.Value(), "26")
}

func TestSeqEmpty(t *testing.T) {
	type args struct {
		items Seq[StringArg]
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.Empty(t, got.items)
}

func TestSeqKVEmpty(t *testing.T) {
	type args struct {
		pairs Seq[KV[string, *StringArg]]
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.Empty(t, got.pairs)
}

type dashArgs struct {
	packs Seq[StringArg]
	sep   Dash
	paths Seq[StringArg]
}

func TestDashSplitsSeqs(t *testing.T) {
	cases := []struct {
		name  string
		args  []string
		packs []string
		paths []string
	}{
		{name: "both sides", args: []string{"a", "b", "--", "c", "d"}, packs: []string{"a", "b"}, paths: []string{"c", "d"}},
		{name: "empty first", args: []string{"--", "."}, packs: []string{}, paths: []string{"."}},
		{name: "empty second", args: []string{"a", "--"}, packs: []string{"a"}, paths: []string{}},
		{name: "dash looking after", args: []string{"pack", "--", "-still", "pos"}, packs: []string{"pack"}, paths: []string{"-still", "pos"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse[dashArgs](tc.args...)
			require.NoError(t, err)
			assert.Equal(t, Values(got.packs), tc.packs)
			assert.Equal(t, Values(got.paths), tc.paths)
		})
	}
}

func TestTwoRestsSplitByDash(t *testing.T) {
	type args struct {
		a []StringArg
		b []StringArg
	}
	got, err := Parse[args]("x", "--", "y", "z")
	require.NoError(t, err)
	assert.Equal(t, Values(got.a), []string{"x"})
	assert.Equal(t, Values(got.b), []string{"y", "z"})
}

func TestTwoSeqsSplitByDash(t *testing.T) {
	type args struct {
		a Seq[StringArg]
		b Seq[StringArg]
	}
	got, err := Parse[args]("x", "--", "y")
	require.NoError(t, err)
	assert.Equal(t, Values(got.a), []string{"x"})
	assert.Equal(t, Values(got.b), []string{"y"})
}

func TestAlgebraErrors(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
		want error
	}{
		{
			name: "seq kv odd",
			run: func() error {
				_, err := Parse[struct{ pairs Seq[KV[string, *StringArg]] }]("name", "lucas", "age")
				return err
			},
			want: ErrMissingValue,
		},
		{
			name: "product missing",
			run: func() error {
				_, err := Parse[struct{ pair KV[string, *StringArg] }]()
				return err
			},
			want: ErrMissingValue,
		},
		{
			name: "product incomplete",
			run: func() error {
				_, err := Parse[struct{ pair KV[string, *StringArg] }]("name")
				return err
			},
			want: ErrMissingValue,
		},
		{
			name: "array short",
			run: func() error {
				_, err := Parse[struct{ pair [2]StringArg }]("a")
				return err
			},
			want: ErrMissingValue,
		},
		{
			name: "array extra",
			run: func() error {
				_, err := Parse[struct{ pair [2]StringArg }]("a", "b", "c")
				return err
			},
			want: ErrInvalidArgument,
		},
		{
			name: "dash missing",
			run: func() error {
				_, err := Parse[dashArgs]("a", "b")
				return err
			},
			want: ErrMissingValue,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIs(t, tc.run(), tc.want)
		})
	}
}

func TestOptionalStringAbsent(t *testing.T) {
	type args struct {
		name *StringArg
		rest []StringArg
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.Nil(t, got.name)
	assert.Empty(t, got.rest)
}

func TestOptionalProduct(t *testing.T) {
	type args struct {
		pair *KV[string, *StringArg]
		rest []StringArg
	}
	t.Run("absent", func(t *testing.T) {
		got, err := Parse[args]()
		require.NoError(t, err)
		assert.Nil(t, got.pair)
		assert.Empty(t, got.rest)
	})
	t.Run("present", func(t *testing.T) {
		got, err := Parse[args]("name", "lucas", "x")
		require.NoError(t, err)
		require.NotNil(t, got.pair)
		assert.Equal(t, got.pair.K.Value(), "name")
		assert.Equal(t, got.pair.V.Value(), "lucas")
		assert.Equal(t, Values(got.rest), []string{"x"})
	})
}

func TestTaggedProductRepeat(t *testing.T) {
	type args struct {
		pairs []KV[string, *StringArg] `long:"pair"`
	}
	got, err := Parse[args]("--pair", "name", "lucas", "--pair", "age", "26")
	require.NoError(t, err)
	require.Len(t, got.pairs, 2)
	assert.Equal(t, got.pairs[0].K.Value(), "name")
	assert.Equal(t, got.pairs[0].V.Value(), "lucas")
	assert.Equal(t, got.pairs[1].K.Value(), "age")
	assert.Equal(t, got.pairs[1].V.Value(), "26")
}

func TestOptionalDash(t *testing.T) {
	type args struct {
		packs Seq[StringArg]
		sep   *Dash
		paths Seq[StringArg]
	}
	t.Run("present", func(t *testing.T) {
		got, err := Parse[args]("a", "--", "b")
		require.NoError(t, err)
		assert.Equal(t, Values(got.packs), []string{"a"})
		require.NotNil(t, got.sep)
		assert.Equal(t, Values(got.paths), []string{"b"})
	})
	t.Run("absent", func(t *testing.T) {
		got, err := Parse[args]("a", "b")
		require.NoError(t, err)
		assert.Equal(t, Values(got.packs), []string{"a", "b"})
		assert.Nil(t, got.sep)
		assert.Empty(t, got.paths)
	})
}

func TestFlagWithSeqKV(t *testing.T) {
	type args struct {
		force Flag `long:"force" short:"f"`
		pairs Seq[KV[string, *StringArg]]
	}
	got, err := Parse[args]("--force", "name", "lucas")
	require.NoError(t, err)
	assert.True(t, got.force.Value())
	require.Len(t, got.pairs, 1)
	assert.Equal(t, got.pairs[0].K.Value(), "name")
	assert.Equal(t, got.pairs[0].V.Value(), "lucas")
}
