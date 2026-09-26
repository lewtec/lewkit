package treesitter

import (
	"context"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

type fake struct {
	id     string
	weight int
	langs  map[string]string
	ext    map[string]string
}

func (f fake) ID() string                               { return f.id }
func (f fake) Name() string                             { return f.id }
func (f fake) Weight() int                              { return f.weight }
func (f fake) CheckCompatibility(context.Context) error { return nil }
func (f fake) New(context.Context) (Driver, error)      { return f, nil }

func (f fake) Names() []string { return slices.Sorted(maps.Keys(f.langs)) }

func (f fake) Language(name string) (Language, bool) {
	backend, ok := f.langs[name]
	if !ok {
		return nil, false
	}
	return marker{name: name, backend: backend}, true
}

func (f fake) ByExtension(filename string) (Language, bool) {
	name, ok := f.ext[filename]
	if !ok {
		return nil, false
	}
	return f.Language(name)
}

type marker struct{ name, backend string }

func (m marker) Name() string { return m.name }

func (m marker) Parse([]byte) (*Tree, error) {
	return NewTree(markerNode{typ: m.backend + ":" + m.name}), nil
}

type markerNode struct{ typ string }

func (m markerNode) Type() string                    { return m.typ }
func (m markerNode) StartByte() uint32               { return 0 }
func (m markerNode) EndByte() uint32                 { return 0 }
func (m markerNode) ChildCount() uint32              { return 0 }
func (m markerNode) Child(uint32) Node               { return m }
func (m markerNode) NamedChildCount() uint32         { return 0 }
func (m markerNode) NamedChild(uint32) Node          { return m }
func (m markerNode) FieldNameForChild(uint32) string { return "" }
func (m markerNode) IsNull() bool                    { return m.typ == "" }
func (m markerNode) IsNamed() bool                   { return false }
func (m markerNode) IsExtra() bool                   { return false }
func (m markerNode) IsError() bool                   { return false }
func (m markerNode) HasError() bool                  { return false }
func (m markerNode) String() string                  { return m.typ }

var registerFakes = sync.OnceFunc(func() {
	driver.Register[Driver](fake{
		id: "treesitter_test_high", weight: 60,
		langs: map[string]string{"json": "high"},
		ext:   map[string]string{"a.json": "json"},
	})
	driver.Register[Driver](fake{
		id: "treesitter_test_low", weight: 40,
		langs: map[string]string{"json": "low", "cobol": "low", "go": "low"},
		ext:   map[string]string{"a.json": "json", "main.go": "go"},
	})
})

func TestHigherWeightWins(t *testing.T) {
	registerFakes()
	lang, err := Get(t.Context(), "json")
	require.NoError(t, err)
	require.Equal(t, "json", lang.Name())
	tree, err := lang.Parse(nil)
	require.NoError(t, err)
	require.Equal(t, "high:json", tree.RootNode().Type())
}

func TestOtherEngineFillsGap(t *testing.T) {
	registerFakes()
	lang, err := Get(t.Context(), "cobol")
	require.NoError(t, err)
	tree, err := Parse(t.Context(), "cobol", []byte("x"))
	require.NoError(t, err)
	require.Equal(t, "low:cobol", tree.RootNode().Type())
	require.Equal(t, "cobol", lang.Name())
}

func TestUnknownLanguage(t *testing.T) {
	registerFakes()
	_, err := Get(t.Context(), "nope")
	require.ErrorIs(t, err, ErrUnknown)
	_, err = Get(t.Context(), "")
	require.ErrorIs(t, err, ErrUnknown)
}

func TestNamesUnion(t *testing.T) {
	registerFakes()
	names, err := Names(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"cobol", "go", "json"}, names)
}

func TestForFileUsesWeight(t *testing.T) {
	registerFakes()
	lang, err := ForFile(t.Context(), "a.json")
	require.NoError(t, err)
	tree, err := lang.Parse(nil)
	require.NoError(t, err)
	require.Equal(t, "high:json", tree.RootNode().Type())

	lang, err = ForFile(t.Context(), "main.go")
	require.NoError(t, err)
	require.Equal(t, "go", lang.Name())
}
