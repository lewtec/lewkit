package vulkan

import (
	"context"
	"embed"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/singleton"
)

//go:embed shader/fill.vert shader/fill.frag shader/ink.vert shader/ink.frag
var drawShaders embed.FS

// drawModules is the fill and ink SPIR-V for one swapchain draw.
type drawModules struct {
	vert, frag, inkVert, inkFrag []byte
}

// drawCode compiles the swapchain fill and ink shaders once.
// The GLSL source is shader/. The compiler is the wasm glslang in x/ffi/wasm/glsl.
var drawCode = singleton.NewSingleton(func(ctx context.Context) (drawModules, error) {
	var code drawModules
	steps := []struct {
		stage glsl.Stage
		name  string
		dst   *[]byte
	}{
		{glsl.StageVertex, "shader/fill.vert", &code.vert},
		{glsl.StageFragment, "shader/fill.frag", &code.frag},
		{glsl.StageVertex, "shader/ink.vert", &code.inkVert},
		{glsl.StageFragment, "shader/ink.frag", &code.inkFrag},
	}
	for _, shader := range steps {
		src, err := drawShaders.ReadFile(shader.name)
		if err != nil {
			return drawModules{}, err
		}
		*shader.dst, err = glsl.CompileStage(ctx, shader.stage, src)
		if err != nil {
			return drawModules{}, err
		}
	}
	return code, nil
})
