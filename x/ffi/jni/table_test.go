package jni

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestFunctionTableFollowsEnvPointer(t *testing.T) {
	env, table := newTableFixture()
	table[jniVersion] = 0x1111
	table[6] = 0x2222
	env[0] = uintptr(unsafe.Pointer(table)) | (0xb4 << 56)
	env[6] = 0x3333
	env[19] = 0xb40000abcdef
	envAddr := uintptr(unsafe.Pointer(env))
	tableAddr := uintptr(unsafe.Pointer(table))

	self, tab, ok := functionTable(envAddr, loadWord)
	require.True(t, ok)
	require.Equal(t, envAddr, self)
	require.Equal(t, tableAddr, tab)
}

func TestFunctionTableKeepsEnvTag(t *testing.T) {
	const (
		tableAddr = uintptr(0x1000)
		envAddr   = uintptr(0x2000)
	)
	step := unsafe.Sizeof(uintptr(0))
	mem := map[uintptr]uintptr{
		tableAddr + 4*step: 0x1111,
		tableAddr + 6*step: 0x2222,
		envAddr:            tableAddr | (0xb4 << 56),
		envAddr + 6*step:   0x3333,
		envAddr + 19*step:  0xb40000abcdef,
	}
	raw := envAddr | (0xb4 << 56)
	load := func(p uintptr) uintptr { return mem[p] }

	self, tab, ok := functionTable(raw, load)
	require.True(t, ok)
	require.Equal(t, raw, self)
	require.Equal(t, tableAddr, tab)
}

func TestFunctionTableAcceptsBareTable(t *testing.T) {
	_, table := newTableFixture()
	table[jniVersion] = 0x1111
	raw := uintptr(unsafe.Pointer(table))

	self, tab, ok := functionTable(raw, loadWord)
	require.True(t, ok)
	require.Equal(t, raw, self)
	require.Equal(t, raw, tab)
}

func TestFunctionTableRejectsLocalRefs(t *testing.T) {
	env, locals := newTableFixture()
	for i := range locals {
		locals[i] = uintptr(i + 1)
	}
	env[0] = uintptr(unsafe.Pointer(locals))

	_, _, ok := functionTable(uintptr(unsafe.Pointer(env)), loadWord)
	require.False(t, ok)
	_, _, ok = functionTable(0, loadWord)
	require.False(t, ok)
}

type tableFixture struct {
	env   [24]uintptr
	table [8]uintptr
}

func newTableFixture() (env *[24]uintptr, table *[8]uintptr) {
	f := &tableFixture{}
	return &f.env, &f.table
}

func loadWord(p uintptr) uintptr {
	return *(*uintptr)(unsafe.Pointer(p))
}
