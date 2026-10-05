package windows

import (
	"testing"
	"unsafe"
)

func TestInfoFlags(t *testing.T) {
	if infoFlags("critical") != niifError {
		t.Fatalf("critical")
	}
	if infoFlags("low") != niifInfo|niifNoSound {
		t.Fatalf("low")
	}
	if infoFlags("normal") != niifInfo || infoFlags("") != niifInfo {
		t.Fatalf("normal")
	}
}

func TestNotifyIconLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip()
	}
	var data notifyIconData
	if unsafe.Sizeof(data) != 952 {
		t.Fatalf("size %d", unsafe.Sizeof(data))
	}
	if unsafe.Offsetof(data.infoFlags) != 948 {
		t.Fatalf("infoFlags %d", unsafe.Offsetof(data.infoFlags))
	}
}
