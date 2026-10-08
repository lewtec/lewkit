package vulkan

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/ndeval/spell"
	"github.com/lewtec/lewkit/x/ndarray"
)

// GLSL spells a non-finite float with uintBitsToFloat.
var glslBody = spell.Dialect{
	NonFinite: "uintBitsToFloat(%du)",
	MinInt32:  "int(0x80000000u)",
}

// glslSource renders the kernel schedule as a Vulkan compute shader.
func glslSource(k *ndarray.Kernel) (string, error) {
	if k == nil {
		return "", ndarray.ErrOp
	}
	code, err := k.Code()
	if err != nil {
		return "", err
	}
	return glslOf(code), nil
}

func glslOf(c ndarray.Code) string {
	var b strings.Builder
	b.WriteString("#version 450\n")
	b.WriteString("layout(local_size_x = ")
	b.WriteString(strconv.Itoa(c.Threads))
	b.WriteString(") in;\n")
	b.WriteString("layout(push_constant) uniform Push { uint n; uint d0; uint d1; uint d2; uint d3; };\n")
	fmt.Fprintf(&b, "layout(set = 0, binding = 0) buffer Out { %s o[]; };\n", spell.Scalar(c.Out))
	for i, dt := range c.In {
		fmt.Fprintf(&b, "layout(set = 0, binding = %d) buffer In%d { %s x%d[]; };\n", i+1, i+1, spell.Scalar(dt), i+1)
	}
	b.WriteString("void main() {\n")
	b.WriteString("    uint gi = gl_GlobalInvocationID.x;\n")
	spell.WriteBody(&b, c, glslBody)
	b.WriteString("}\n")
	return b.String()
}
