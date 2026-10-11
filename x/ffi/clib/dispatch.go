//go:build ((linux || android) && (amd64 || arm64 || 386 || arm)) || (windows && (amd64 || arm64 || 386)) || (darwin && !ios && (amd64 || arm64))

package clib

import "sync/atomic"

func dispatch(slot *Slot) {
	defer func() {
		if recover() != nil {
			slot.Ret = 0
			slot.RetX = 0
			slot.Kind = KindInt
		}
		atomic.StoreUint32(&slot.Done, 1)
		wake(&slot.Done)
	}()
	h := handler(slot.ID)
	if h == nil {
		return
	}
	slot.Ret, slot.RetX, slot.Kind = h(&slot.Args, &slot.XMM)
}
