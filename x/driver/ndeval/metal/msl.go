package metal

import (
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/driver/ndeval/spell"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Metal spells a non-finite float with as_type. The int32 minimum matches GLSL.
var metalBody = spell.Dialect{
	NonFinite: "as_type<float>(%du)",
	MinInt32:  "int(0x80000000u)",
}

// mslSource renders the kernel schedule as one Metal compute kernel.
func mslSource(k *ndarray.Kernel) (string, int, error) {
	if k == nil {
		return "", 0, ndarray.ErrOp
	}
	code, err := k.Code()
	if err != nil {
		return "", 0, err
	}
	return mslOf(code), code.Threads, nil
}

func mslOf(c ndarray.Code) string {
	var b strings.Builder
	b.WriteString("#include <metal_stdlib>\nusing namespace metal;\n\n")
	b.WriteString("struct Push {\n    uint n;\n    uint d0;\n    uint d1;\n    uint d2;\n    uint d3;\n};\n\n")
	b.WriteString("kernel void ndeval(\n")
	fmt.Fprintf(&b, "    device %s* o [[buffer(0)]],\n", spell.Scalar(c.Out))
	for i, dt := range c.In {
		fmt.Fprintf(&b, "    device %s* x%d [[buffer(%d)]],\n", spell.Scalar(dt), i+1, i+1)
	}
	fmt.Fprintf(&b, "    constant Push& push [[buffer(%d)]],\n", 1+len(c.In))
	b.WriteString("    uint gid [[thread_position_in_grid]])\n{\n")
	b.WriteString("    uint n = push.n;\n")
	b.WriteString("    uint d0 = push.d0;\n")
	b.WriteString("    uint d1 = push.d1;\n")
	b.WriteString("    uint d2 = push.d2;\n")
	b.WriteString("    uint d3 = push.d3;\n")
	b.WriteString("    uint gi = gid;\n")
	spell.WriteBody(&b, c, metalBody)
	b.WriteString("}\n")
	return b.String()
}
