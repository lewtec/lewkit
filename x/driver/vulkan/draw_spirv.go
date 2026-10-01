package vulkan

import (
	"context"
	"embed"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
)

//go:embed shader/fill.vert shader/fill.frag shader/ink.vert shader/ink.frag
var drawShaders embed.FS

var drawSPIRV struct {
	sync.Mutex
	done                         bool
	vert, frag, inkVert, inkFrag []byte
	err                          error
}

// drawCode compiles the swapchain fill and ink shaders once.
// The GLSL source is shader/. The compiler is the wasm glslang in x/ffi/wasm/glsl.
func drawCode(ctx context.Context) (vert, frag, inkVert, inkFrag []byte, err error) {
	drawSPIRV.Lock()
	defer drawSPIRV.Unlock()
	if drawSPIRV.done {
		return drawSPIRV.vert, drawSPIRV.frag, drawSPIRV.inkVert, drawSPIRV.inkFrag, drawSPIRV.err
	}
	type step struct {
		stage glsl.Stage
		name  string
		dst   *[]byte
	}
	for _, shader := range []step{
		{glsl.StageVertex, "shader/fill.vert", &drawSPIRV.vert},
		{glsl.StageFragment, "shader/fill.frag", &drawSPIRV.frag},
		{glsl.StageVertex, "shader/ink.vert", &drawSPIRV.inkVert},
		{glsl.StageFragment, "shader/ink.frag", &drawSPIRV.inkFrag},
	} {
		var src []byte
		src, err = drawShaders.ReadFile(shader.name)
		if err != nil {
			break
		}
		*shader.dst, err = glsl.CompileStage(ctx, shader.stage, src)
		if err != nil {
			break
		}
	}
	drawSPIRV.err, drawSPIRV.done = err, true
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return drawSPIRV.vert, drawSPIRV.frag, drawSPIRV.inkVert, drawSPIRV.inkFrag, nil
}
