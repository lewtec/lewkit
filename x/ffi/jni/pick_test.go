package jni

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassify(t *testing.T) {
	v, err := classify(nil)
	require.NoError(t, err)
	require.Equal(t, kindNil, v.kind)

	v, err = classify("hi")
	require.NoError(t, err)
	require.Equal(t, kindString, v.kind)
	require.Equal(t, "hi", v.text)

	v, err = classify(byte(7))
	require.NoError(t, err)
	require.Equal(t, kindInt, v.kind)
	require.Equal(t, int32(7), v.i32)

	v, err = classify(int64(math.MaxInt32) + 1)
	require.NoError(t, err)
	require.Equal(t, kindLong, v.kind)

	v, err = classify(&Ref{})
	require.NoError(t, err)
	require.Equal(t, kindNil, v.kind)

	v, err = classify(&Ref{ptr: 4, class: "java.lang.String"})
	require.NoError(t, err)
	require.Equal(t, kindRef, v.kind)
	require.Equal(t, "java.lang.String", v.class)

	_, err = classify(struct{}{})
	require.Error(t, err)
}

func TestKeepMemberSkipsBridge(t *testing.T) {
	require.False(t, keepMember(modBridge|modStatic, true))
	require.True(t, keepMember(modStatic, true))
	require.False(t, keepMember(modStatic, false))
	require.True(t, keepMember(0, false))
}

func TestPickHostMethods(t *testing.T) {
	cands := []candidate{
		{name: "dataDir", static: true, ret: "java.lang.String"},
		{name: "ready", static: true, params: []string{"java.lang.String"}, ret: "void"},
		{name: "openSurface", static: true, ret: "void"},
		{name: "fail", static: true, params: []string{"java.lang.String"}, ret: "void"},
		{name: "ready", static: false, params: []string{"java.lang.String"}, ret: "void"},
	}
	i, err := pick(cands, selector{name: "dataDir", static: true})
	require.NoError(t, err)
	require.Equal(t, 0, i)

	i, err = pick(cands, selector{name: "openSurface", static: true})
	require.NoError(t, err)
	require.Equal(t, "openSurface", cands[i].name)

	i, err = pick(cands, selector{name: "ready", static: true, args: []arg{{kind: kindString}}})
	require.NoError(t, err)
	require.True(t, cands[i].static)

	_, err = pick(cands, selector{name: "missing", static: true})
	require.Error(t, err)
}

func TestPickOverload(t *testing.T) {
	cands := []candidate{
		{name: "foo", static: true, params: []string{"long"}},
		{name: "foo", static: true, params: []string{"int"}},
		{name: "foo", static: true, params: []string{"java.lang.String"}},
		{name: "foo", static: true, params: []string{"java.lang.Object"}},
	}
	i, err := pick(cands, selector{name: "foo", static: true, args: []arg{{kind: kindInt}}})
	require.NoError(t, err)
	require.Equal(t, "int", cands[i].params[0])

	i, err = pick(cands, selector{name: "foo", static: true, args: []arg{{kind: kindLong}}})
	require.NoError(t, err)
	require.Equal(t, "long", cands[i].params[0])

	i, err = pick(cands, selector{name: "foo", static: true, args: []arg{{kind: kindString}}})
	require.NoError(t, err)
	require.Equal(t, "java.lang.String", cands[i].params[0])

	_, err = pick(cands, selector{name: "foo", static: true, args: []arg{{kind: kindBool}}})
	require.Error(t, err)

	_, err = pick(cands, selector{name: "foo", static: true, args: []arg{{kind: kindNil}}})
	require.Error(t, err)
}

func TestPickRefPrefersExactClass(t *testing.T) {
	cands := []candidate{
		{name: "add", static: true, params: []string{"java.lang.Object"}},
		{name: "add", static: true, params: []string{"java.lang.String"}},
		{name: "add", static: true, params: []string{"java.lang.CharSequence"}},
	}
	fit := map[string]bool{
		"java.lang.Object":       true,
		"java.lang.CharSequence": true,
		"java.lang.String":       true,
	}
	i, err := pick(cands, selector{name: "add", static: true, args: []arg{{
		kind:       kindRef,
		class:      "java.lang.String",
		assignable: func(param string) bool { return fit[param] },
	}}})
	require.NoError(t, err)
	require.Equal(t, "java.lang.String", cands[i].params[0])
}

func TestPickAmbiguousInterfaces(t *testing.T) {
	cands := []candidate{
		{name: "add", static: true, params: []string{"java.lang.Runnable"}},
		{name: "add", static: true, params: []string{"java.lang.Readable"}},
	}
	_, err := pick(cands, selector{name: "add", static: true, args: []arg{{
		kind:       kindRef,
		class:      "app.Both",
		assignable: func(string) bool { return true },
	}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "ambiguous")
}

func TestClassNameSlashes(t *testing.T) {
	require.Equal(t, "lewkit.Host", dotted("lewkit/Host"))
	require.Equal(t, "lewkit/Host$Inner", slashed("lewkit.Host$Inner"))
	require.Equal(t, "java/lang/String", slashed("java.lang.String"))
}
