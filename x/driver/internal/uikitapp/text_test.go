package uikitapp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAskTitle(t *testing.T) {
	require.Equal(t, " Saved ", askTitle(kindAlert, " Saved "))
	require.Equal(t, "Confirm", askTitle(kindConfirm, " "))
	require.Equal(t, "Input", askTitle(kindPrompt, ""))
	require.Equal(t, "Choose", askTitle(kindChoose, ""))
	require.Equal(t, "Error", askTitle(kindAlert, ""))
}

func TestAskLines(t *testing.T) {
	require.Equal(t, []string{"one", "two"}, askLines("one\n\ntwo"))
}

func TestSafeName(t *testing.T) {
	require.Equal(t, "passwd", safeName("../etc/passwd"))
	require.Equal(t, "untitled", safeName(" "))
	require.Equal(t, "untitled", safeName(".."))
}

func TestBatteryRead(t *testing.T) {
	status, level := batteryRead(2, 0.4)
	require.Equal(t, "Charging", status)
	require.Equal(t, 40, level)
	status, level = batteryRead(0, -1)
	require.Equal(t, "Unknown", status)
	require.Equal(t, -1, level)
	require.Equal(t, "dark", appearanceName(2))
	require.Equal(t, "light", appearanceName(1))
	require.Equal(t, "light", appearanceName(0))
}
