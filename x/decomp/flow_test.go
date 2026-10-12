package decomp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInspectCallAndBranch(t *testing.T) {
	call := inspect("call", "0x1006")
	assert.Equal(t, kindCall, call.kind)
	assert.True(t, call.direct)
	assert.Equal(t, uint64(0x1006), call.target)

	branch := inspect("je", "0x1002")
	assert.Equal(t, kindBranch, branch.kind)
	assert.Equal(t, uint64(0x1002), branch.target)

	indirect := inspect("call", "rax")
	assert.Equal(t, kindCall, indirect.kind)
	assert.False(t, indirect.direct)

	ret := inspect("ret", "")
	assert.Equal(t, kindReturn, ret.kind)
}

func TestRipDisplacement(t *testing.T) {
	displacement, ok := ripDisplacement("qword ptr [rip + 0xff9]")
	assert.True(t, ok)
	assert.Equal(t, int64(0xff9), displacement)

	displacement, ok = ripDisplacement("qword ptr [rip - 0x10]")
	assert.True(t, ok)
	assert.Equal(t, int64(-0x10), displacement)

	addresses := referencedAddresses("qword ptr [rip + 0xff9]", 0x1007)
	assert.Equal(t, []uint64{0x2000, 0xff9}, addresses)

	control := inspect("lea", "rdi, [rip + 0xff9]")
	assert.Equal(t, kindPlain, control.kind)
	assert.False(t, control.direct)
}
