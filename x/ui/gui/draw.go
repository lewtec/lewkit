package gui

import (
	"context"
	"encoding/binary"
	"math"
	"sync"

	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/ndarray"
)

const fillVertGLSL = `#version 450
layout(set = 0, binding = 0) readonly buffer Inst { vec4 data[]; };
layout(push_constant) uniform Push { vec2 extent; int swapRB; } push;
layout(location = 0) flat out int instance;
void main() {
    instance = gl_InstanceIndex;
    vec4 box = data[instance * 4];
    vec2 unit = vec2(float(gl_VertexIndex & 1), float(gl_VertexIndex >> 1));
    vec2 pos = box.xy + unit * box.zw;
    vec2 ndc = vec2(pos.x / push.extent.x * 2.0 - 1.0, pos.y / push.extent.y * 2.0 - 1.0);
    gl_Position = vec4(ndc, float(push.swapRB) * 0.0, 1.0);
}
`

const fillFragGLSL = `#version 450
layout(set = 0, binding = 0) readonly buffer Inst { vec4 data[]; };
layout(push_constant) uniform Push { vec2 extent; int swapRB; } push;
layout(location = 0) flat in int instance;
layout(location = 0) out vec4 outColor;
void main() {
    int base = instance * 4;
    vec4 box = data[base];
    vec4 color = data[base + 1];
    vec4 rad = data[base + 2];
    float clipH = data[base + 3].x;
    vec2 pixel = gl_FragCoord.xy;
    float halfv = 0.5;
    float localX = pixel.x - box.x + halfv - box.z * halfv;
    float localY = pixel.y - box.y + halfv - box.w * halfv;
    float halfW = box.z * halfv;
    float halfH = box.w * halfv;
    float radius = rad.x;
    radius = radius < halfW ? radius : halfW;
    radius = radius < halfH ? radius : halfH;
    float cx = abs(localX) - halfW + radius;
    float cy = abs(localY) - halfH + radius;
    float ox = max(cx, 0.0);
    float oy = max(cy, 0.0);
    float cornerMax = max(cx, cy);
    float inside = cornerMax < 0.0 ? cornerMax : 0.0;
    float dist = inside + sqrt(ox * ox + oy * oy) - radius;
    float cover = dist < halfv ? 1.0 : 0.0;
    if (rad.w >= halfv) {
        bool insideX = pixel.x >= rad.y && pixel.x < rad.y + rad.w;
        bool insideY = pixel.y >= rad.z && pixel.y < rad.z + clipH;
        if (!insideX || !insideY) cover = 0.0;
    }
    float a = cover * color.a * (1.0 / 255.0);
    vec3 rgb = color.rgb * (1.0 / 255.0) * a;
    if (push.swapRB != 0) rgb = rgb.bgr;
    outColor = vec4(rgb, a);
}
`

const inkVertGLSL = `#version 450
layout(push_constant) uniform Push { ivec2 size; int swapRB; } push;
void main() {
    vec2 unit = vec2(float(gl_VertexIndex & 1), float(gl_VertexIndex >> 1));
    gl_Position = vec4(unit * 2.0 - 1.0, float(push.swapRB) * 0.0, 1.0);
}
`

const inkFragGLSL = `#version 450
layout(set = 0, binding = 0) readonly buffer Pix { uint words[]; };
layout(push_constant) uniform Push { ivec2 size; int swapRB; } push;
layout(location = 0) out vec4 outColor;
void main() {
    ivec2 c = ivec2(gl_FragCoord.xy);
    if (c.x < 0 || c.y < 0 || c.x >= push.size.x || c.y >= push.size.y) {
        outColor = vec4(0.0);
        return;
    }
    uint w = words[c.y * push.size.x + c.x];
    float r = float(w & 255u) / 255.0;
    float g = float((w >> 8u) & 255u) / 255.0;
    float b = float((w >> 16u) & 255u) / 255.0;
    float a = float((w >> 24u) & 255u) / 255.0;
    if (r == 0.0 && g == 0.0 && b == 0.0 && a == 0.0) {
        outColor = vec4(0.0);
        return;
    }
    if (a == 0.0) a = 1.0;
    vec3 rgb = vec3(r, g, b) * a;
    if (push.swapRB != 0) rgb = rgb.bgr;
    outColor = vec4(rgb, a);
}
`

var drawSPIRV struct {
	sync.Mutex
	done                         bool
	vert, frag, inkVert, inkFrag []byte
	err                          error
}

func drawCode(ctx context.Context) (vert, frag, inkVert, inkFrag []byte, err error) {
	drawSPIRV.Lock()
	defer drawSPIRV.Unlock()
	if drawSPIRV.done {
		return drawSPIRV.vert, drawSPIRV.frag, drawSPIRV.inkVert, drawSPIRV.inkFrag, drawSPIRV.err
	}
	drawSPIRV.vert, err = glsl.CompileStage(ctx, glsl.StageVertex, []byte(fillVertGLSL))
	if err != nil {
		drawSPIRV.err, drawSPIRV.done = err, true
		return nil, nil, nil, nil, err
	}
	drawSPIRV.frag, err = glsl.CompileStage(ctx, glsl.StageFragment, []byte(fillFragGLSL))
	if err != nil {
		drawSPIRV.err, drawSPIRV.done = err, true
		return nil, nil, nil, nil, err
	}
	drawSPIRV.inkVert, err = glsl.CompileStage(ctx, glsl.StageVertex, []byte(inkVertGLSL))
	if err != nil {
		drawSPIRV.err, drawSPIRV.done = err, true
		return nil, nil, nil, nil, err
	}
	drawSPIRV.inkFrag, err = glsl.CompileStage(ctx, glsl.StageFragment, []byte(inkFragGLSL))
	drawSPIRV.err, drawSPIRV.done = err, true
	return drawSPIRV.vert, drawSPIRV.frag, drawSPIRV.inkVert, drawSPIRV.inkFrag, err
}

// drawFills paints recorded fills and optional RGBA8 ink. It does not run the fused kernel.
func drawFills(ctx context.Context, screen vulkan.Screen, fills []Draw, ink []byte, width, height int) error {
	if screen == nil || width < 1 || height < 1 {
		return ndarray.ErrShape
	}
	vert, frag, inkVert, inkFrag, err := drawCode(ctx)
	if err != nil {
		return err
	}
	raw := make([]byte, len(fills)*64)
	for i, fill := range fills {
		putFill(raw[i*64:(i+1)*64], fill)
	}
	return screen.Draw(raw, ink, width, height, vert, frag, inkVert, inkFrag)
}

func putFill(dst []byte, fill Draw) {
	vals := [16]float32{
		fill.X, fill.Y, fill.Width, fill.Height,
		fill.Red, fill.Green, fill.Blue, fill.Alpha,
		fill.Radius, fill.ClipX, fill.ClipY, fill.ClipWidth,
		fill.ClipHeight, 0, 0, 0,
	}
	for i, v := range vals {
		binary.LittleEndian.PutUint32(dst[i*4:], math.Float32bits(v))
	}
}
