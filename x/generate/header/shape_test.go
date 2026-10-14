package header

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShapeMatchesKind(t *testing.T) {
	pairs := []struct {
		k kind
		s Shape
	}{
		{kindInvalid, ShapeInvalid},
		{kindVoid, ShapeVoid},
		{kindBool, ShapeBool},
		{kindSInt, ShapeSInt},
		{kindUInt, ShapeUInt},
		{kindFloat, ShapeFloat},
		{kindPtr, ShapePtr},
		{kindArray, ShapeArray},
		{kindFunc, ShapeFunc},
		{kindStruct, ShapeStruct},
		{kindUnion, ShapeUnion},
		{kindEnum, ShapeEnum},
		{kindAlias, ShapeAlias},
	}
	if len(pairs) != int(kindAlias)+1 {
		require.FailNowf(t, "", "kind gained a value; update Shape")
	}
	for _, pair := range pairs {
		if Shape(pair.k) != pair.s {
			require.FailNowf(t, "", "kind %d is shape %d, want %d", pair.k, Shape(pair.k), pair.s)
		}
	}
}
