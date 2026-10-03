//go:build windows

package windows

import (
	"fmt"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/native"
)

var procPower = native.ProcOf("kernel32.dll", "GetSystemPowerStatus")

type systemPowerStatus struct {
	ac       byte
	flag     byte
	percent  byte
	reserved byte
	life     uint32
	full     uint32
}

func readPower() (byte, byte, byte, error) {
	var status systemPowerStatus
	ok, _, err := procPower.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return 0, 0, 0, fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return status.ac, status.flag, status.percent, nil
}
