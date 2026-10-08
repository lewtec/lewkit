package d3d12

import (
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver/ndeval/spell"
	"github.com/lewtec/lewkit/x/ndarray"
)

// cs_5_0 rejects int(bool). A non-finite float is asfloat, and the most
// negative int32 is asint.
var hlslBody = spell.Dialect{
	NonFinite:    "asfloat(%du)",
	MinInt32:     "asint(0x80000000u)",
	IntRelSelect: true,
}

// hlslSource renders the kernel schedule as one HLSL compute kernel.
// n and d0..d3 are the cbuffer members, so the body does not redeclare them.
func hlslSource(k *ndarray.Kernel) (string, int, error) {
	if k == nil {
		return "", 0, ndarray.ErrOp
	}
	code, err := k.Code()
	if err != nil {
		return "", 0, err
	}
	return hlslOf(code), code.Threads, nil
}

func hlslOf(c ndarray.Code) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RWStructuredBuffer<%s> o : register(u0);\n", spell.Scalar(c.Out))
	for i, dt := range c.In {
		fmt.Fprintf(&b, "StructuredBuffer<%s> x%d : register(t%d);\n", spell.Scalar(dt), i+1, i)
	}
	// span is the sixth root constant: groupsX * numthreads.x.
	// A dispatch wider than 65535 groups stacks the rest on Y.
	// tid.y is 0 when that stack is one row, so gid stays tid.x.
	b.WriteString("cbuffer Push : register(b0) {\n")
	b.WriteString("    uint n;\n    uint d0;\n    uint d1;\n    uint d2;\n    uint d3;\n    uint span;\n};\n")
	fmt.Fprintf(&b, "[numthreads(%d, 1, 1)]\n", c.Threads)
	b.WriteString("void ndeval(uint3 tid : SV_DispatchThreadID) {\n")
	b.WriteString("    uint gid = tid.y * span + tid.x;\n")
	b.WriteString("    uint gi = gid;\n")
	spell.WriteBody(&b, c, hlslBody)
	b.WriteString("}\n")
	return b.String()
}
