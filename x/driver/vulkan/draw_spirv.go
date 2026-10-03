package vulkan

//go:generate go run ../../../cmd/lewkit generate shader .

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

// drawCode loads the swapchain fill and ink shaders once.
// The GLSL source is shader/. A registered hash returns SPIR-V.
// A missing hash compiles with glslc or the embedded glslang.
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
