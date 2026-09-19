package main

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriverLabelTypeThenName(t *testing.T) {
	assert.Equal(t, "ndarray_vulkan: Apple M5", driverLabel(driver.DriverStatus{
		ID:   "ndarray_vulkan",
		Name: "Apple M5",
	}))
	assert.Equal(t, "ndarray_cpu", driverLabel(driver.DriverStatus{
		ID:   "ndarray_cpu",
		Name: "ndarray_cpu",
	}))
	assert.Equal(t, "Apple M5 w=50", driverDetail(driver.DriverStatus{
		ID:     "ndarray_vulkan",
		Name:   "Apple M5",
		Weight: 50,
	}))
}

func TestDoctorUsage(t *testing.T) {
	text, err := cmd.Usage[doctorCmd]("lewkit doctor")
	require.NoError(t, err)
	assert.Contains(t, text, "interface => implementation")
}

func TestDoctorPrintsTree(t *testing.T) {
	test.RestoreSlog(t)
	app := cmd.ParseOK[cmd.App[root]](t, "doctor")
	got := test.Stdout(t, func() {
		require.NoError(t, app.Run(t.Context()))
	})
	assert.Contains(t, got, "window.Driver")
	assert.Contains(t, got, "=>")
	assert.Contains(t, got, "window_mem")
	assert.Contains(t, got, "window_mem: Memory")
	assert.Contains(t, got, "w=50")
	assert.True(t, strings.Contains(got, "├ ") || strings.Contains(got, "└ "))
}
