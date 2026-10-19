package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errRead = errors.New("read")

func TestErrScaffoldIsUnsupported(t *testing.T) {
	require.ErrorIs(t, ErrScaffold, ErrUnsupported)
	require.EqualError(t, ErrScaffold, "scaffold")
}

func TestWrapKeepsTextAndParent(t *testing.T) {
	err := Wrap(ErrNotFound, "conda package not found")
	require.ErrorIs(t, err, ErrNotFound)
	require.EqualError(t, err, "conda package not found")
	require.NotErrorIs(t, err, ErrMissing)
	require.NotErrorIs(t, err, ErrInvalid)
	require.ErrorIs(t, errors.Unwrap(err), ErrNotFound)
}

func TestWrapMatchesEveryParentWithoutJoiningText(t *testing.T) {
	err := Wrap(ErrNotFound, "conda package not found", ErrMissing)
	require.EqualError(t, err, "conda package not found")
	require.NotContains(t, err.Error(), ErrMissing.Error())
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorIs(t, err, ErrMissing)
	require.NotErrorIs(t, err, ErrInvalid)

	cause := Wrap(errRead, errRead.Error(), ErrUnavailable)
	require.EqualError(t, cause, errRead.Error())
	require.ErrorIs(t, cause, errRead)
	require.ErrorIs(t, cause, ErrUnavailable)
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name, text, parent string
		collapse           bool
	}{
		{"ErrFile", "file not found", "ErrNotFound", false},
		{"ErrNotFound", "not found", "ErrNotFound", true},
		{"ErrNotFound", "driver not found", "ErrNotFound", false},
		{"ErrNoSuchFileOrDirectory", "no such file or directory", "ErrNotFound", false},
		{"ErrSize", "invalid size", "ErrInvalid", false},
		{"ErrEmbed", "load: bad embed", "ErrInvalid", false},
		{"ErrIsNotAPeFile", "is not a PE file", "ErrInvalid", false},
		{"ErrExport", "missing export", "ErrMissing", false},
		{"ErrHost", "unsupported target", "ErrUnsupported", false},
		{"ErrMakeEvalIsNotSupported", "make: eval is not supported", "ErrUnsupported", false},
		{"ErrIsNotASupportedPeMachine", "is not a supported PE machine", "ErrUnsupported", false},
		{"ErrExist", "already registered", "ErrAlreadyExists", false},
		{"ErrOutDirNotEmpty", "out dir is not empty (use --force)", "ErrAlreadyExists", false},
		{"ErrUnavailable", "gtk unavailable", "ErrUnavailable", false},
		{"ErrCgo", "load: cgo is not available", "ErrUnavailable", false},
		{"ErrClosed", "window closed", "ErrClosed", false},
		{"ErrCanceled", "canceled", "ErrCanceled", true},
		{"ErrCanceled", "tui: canceled", "ErrCanceled", true},
		{"ErrCanceled", "directory canceled", "ErrCanceled", false},
		{"ErrChooseNoItems", "choose: no items", "ErrEmpty", false},
		{"ErrEmptyTarget", "toolchain: empty target", "ErrEmpty", false},
		{"ErrNilContext", "exec: nil context", "ErrNilContext", true},
		{"ErrMakeNilContext", "make: nil context", "ErrNilContext", true},
		{"ErrSyntax", "syntax", "ErrSyntax", true},
		{"ErrCycle", "cycle", "ErrCycle", true},
		{"ErrCycle", "dependency cycle", "ErrCycle", false},
		{"ErrDuplicate", "duplicate step", "ErrDuplicate", false},
		{"ErrIndex", "index out of range", "ErrOutOfRange", false},
		{"ErrOutAppRequired", "out .app path is required (or use --go-only)", "ErrRequired", false},
		{"ErrUnknownDep", "toolchain: unknown dependency", "ErrUnknown", false},
		{"ErrNoVersions", "no versions found", "ErrNotFound", false},
		{"ErrHasNoDynamicSegment", "has no dynamic segment", "ErrNotFound", false},
		{"ErrDecode", "decode", "", false},
		{"ErrShare", "", "", false},
		{"ErrKnown", "known", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/"+tc.text, func(t *testing.T) {
			parent, collapse := Classify(tc.name, tc.text)
			require.Equal(t, tc.parent, parent)
			require.Equal(t, tc.collapse, collapse)
		})
	}
}
