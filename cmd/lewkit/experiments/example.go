package experiments

import (
	"context"
	_ "embed"
	"os"

	"github.com/lewtec/lewkit/x/wasm/glsl"
)

//go:embed example.comp
var exampleComp []byte

func loadShader(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return glsl.Load(ctx, exampleComp)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return glsl.Load(ctx, b)
}
