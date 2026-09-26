package main

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorUsage(t *testing.T) {
	text, err := cmd.Usage[doctorCmd]("lewkit doctor")
	require.NoError(t, err)
	assert.Contains(t, text, "list registered drivers")
	assert.Contains(t, text, "--format")
	assert.Contains(t, text, "--columns")
}

func TestDoctorPrintsTable(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "doctor")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	header := got
	if i := strings.IndexByte(got, '\n'); i >= 0 {
		header = got[:i]
	}
	assert.Less(t, strings.Index(header, "id"), strings.Index(header, "name"))
	assert.Less(t, strings.Index(header, "name"), strings.Index(header, "interface"))
	assert.Contains(t, got, "window.Driver")
	assert.Contains(t, got, "window_mem")
	assert.Contains(t, got, "Memory")
	assert.True(t, strings.Contains(got, "true"))
}

func TestDoctorPrintsJSONL(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "doctor", "--format", "jsonl")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, `"interface":`)
	assert.Contains(t, got, `"id":"window_mem"`)
	assert.NotContains(t, got, "\t")
}

func TestDoctorColumnsFlag(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "doctor", "--columns", "id,name=%q")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	header := got
	if i := strings.IndexByte(got, '\n'); i >= 0 {
		header = got[:i]
	}
	assert.Less(t, strings.Index(header, "id"), strings.Index(header, "name"))
	assert.NotContains(t, header, "interface")
	assert.Contains(t, got, "\"Memory\"")
	assert.NotContains(t, got, "window.Driver")
}
