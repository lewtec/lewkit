package windows

// notifyIconData is NOTIFYICONDATAW through dwInfoFlags.
// On 64-bit Windows that prefix is 952 bytes and dwInfoFlags is at 948.
type notifyIconData struct {
	size            uint32
	window          uintptr
	id              uint32
	flags           uint32
	callbackMessage uint32
	icon            uintptr
	tip             [128]uint16
	state           uint32
	stateMask       uint32
	info            [256]uint16
	timeout         uint32
	infoTitle       [64]uint16
	infoFlags       uint32
}
