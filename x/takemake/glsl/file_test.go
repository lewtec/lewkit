package glsl

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/errs"
	glslang "github.com/lewtec/lewkit/x/ffi/wasm/glsl"
)

// Khronos fixtures under testdata/khronos are Vulkan-Samples shaders:
// https://github.com/KhronosGroup/Vulkan-Samples (Apache-2.0).

func TestAPIRoundTrip(t *testing.T) {
	file := New(Fragment).SetVersion(450).SetProfile(Core)
	file.Extension("GL_EXT_control_flow_attributes", Require)
	file.Precision(Highp, "float")
	file.Out("color", Vec4()).Layout("location", IntLit(0))
	file.Func("main").Returns(Void()).Block(
		Exprs(Op("=", Id("color"), CallOf(Id("vec4"), FloatLit(1), FloatLit(0), FloatLit(0), FloatLit(1)))),
	)
	require.Equal(t, Fragment, file.Stage)
	roundTripFile(t, "api", file)
}

func TestNilMethods(t *testing.T) {
	var file *File
	require.Nil(t, file.SetVersion(450))
	require.Nil(t, file.SetProfile(Core))
	require.Nil(t, file.Extension("GL_ARB_gpu_shader5", Enable))
	require.Nil(t, file.Precision(Highp, "float"))
	file.Uniform("color", Vec4()).Layout("binding", IntLit(0)).Writeonly()
	file.Func("main").Returns(Void()).Block()
	var fn *Func
	require.Nil(t, fn.Returns(Void()))
	require.Nil(t, fn.Param("x", Int()))
	require.Nil(t, fn.Block())
	var decl *Decl
	require.Nil(t, decl.Layout("binding", IntLit(0)))
	require.Nil(t, decl.Writeonly())
	require.Nil(t, decl.SetInstance("block"))
	var item *If
	require.Nil(t, item.SetElse())
	require.Nil(t, item.ElseIf(True()))
}

func TestRepoShaders(t *testing.T) {
	root := moduleRoot(t)
	var n int
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".glsl") {
			return nil
		}
		n++
		src := readText(t, path)
		roundTrip(t, path, src)
		if strings.HasSuffix(path, "example.compute.glsl") {
			made := mustMake(t, src)
			require.Contains(t, made, "4278190080u")
			require.Equal(t, Compute, mustTake(t, src).Stage)
		}
		compileShader(t, path, src, 0)
		return nil
	})
	require.NoError(t, err)
	require.NotZero(t, n)
}

func TestKhronosShaders(t *testing.T) {
	dir := filepath.Join(packageDir(t), "testdata", "khronos")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var n int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		src := readText(t, path)
		if strings.Contains(src, "#include") {
			_, err := Take(strings.NewReader(src))
			require.ErrorIs(t, err, ErrSyntax)
			require.ErrorIs(t, err, errs.ErrSyntax)
			continue
		}
		n++
		roundTrip(t, entry.Name(), src)
		compileShader(t, entry.Name(), src, 0)
	}
	require.GreaterOrEqual(t, n, 8)
}

func TestPresentShader(t *testing.T) {
	root := moduleRoot(t)
	src := rawConst(t, filepath.Join(root, "x/driver/ndeval/vulkan/paint.go"), "presentGLSL")
	require.Contains(t, src, "imageStore")
	roundTrip(t, "presentGLSL", src)
	compileShader(t, "presentGLSL", src, Compute)
}

func TestGeneratorPatterns(t *testing.T) {
	// OpenGL emits loose uniforms. Vulkan GLSL, which the embedded
	// compiler accepts, keeps the same body and moves n into a block.
	desktopGL := `#version 430
layout(local_size_x = 64) in;
uniform uint n;
uniform uint d0;
layout(std430, binding = 0) buffer Out { float o[]; };
layout(std430, binding = 1) buffer In1 { float x1[]; };
void main() {
    uint span = gl_NumWorkGroups.x * gl_WorkGroupSize.x;
    uint gi = gl_GlobalInvocationID.x + gl_GlobalInvocationID.y * span + gl_GlobalInvocationID.z * span * gl_NumWorkGroups.y;
    if (gi >= n) return;
    o[gi] = x1[gi] + uintBitsToFloat(2139095040u);
    o[gi] = o[gi] + float(int(0x80000000u));
}
`
	desktop := `#version 430
layout(local_size_x = 64) in;
layout(std140, binding = 2) uniform Params { uint n; uint d0; };
layout(std430, binding = 0) buffer Out { float o[]; };
layout(std430, binding = 1) buffer In1 { float x1[]; };
void main() {
    uint span = gl_NumWorkGroups.x * gl_WorkGroupSize.x;
    uint gi = gl_GlobalInvocationID.x + gl_GlobalInvocationID.y * span + gl_GlobalInvocationID.z * span * gl_NumWorkGroups.y;
    if (gi >= n) return;
    o[gi] = x1[gi] + uintBitsToFloat(2139095040u);
    o[gi] = o[gi] + float(int(0x80000000u));
}
`
	es := `#version 310 es
precision highp float;
precision highp int;
layout(local_size_x = 8) in;
layout(std140, binding = 1) uniform Params { uint n; };
layout(std430, binding = 0) buffer Out { uint o[]; };
void main() {
    uint gi = gl_GlobalInvocationID.x;
    if (gi >= n) return;
    o[gi] = 0u;
}
`
	roundTrip(t, "desktop-gl", desktopGL)
	roundTrip(t, "desktop", desktop)
	compileShader(t, "desktop.comp", desktop, Compute)
	roundTrip(t, "gles", es)
	compileShader(t, "gles.comp", es, Compute)
}

func TestStatements(t *testing.T) {
	src := `#version 450
void main() {
    int x = 0;
    while (x < 3) {
        x++;
    }
    do {
        x--;
    } while (x > 0);
    switch (x) {
    case 0:
        x = 1;
        break;
    default:
        discard;
    }
    for (; x < 1; x++) {
        ;
    }
    int a = 1, b = 2;
    for (int i = 0, j = 1; i < 2; i++) {
        a = i;
    }
}
`
	roundTrip(t, "statements", src)
	require.Equal(t, Fragment, mustTake(t, src).Stage)
}

func TestSentinels(t *testing.T) {
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrValue)
	require.ErrorIs(t, err, errs.ErrInvalid)

	err = (*File)(nil).Make(io.Discard)
	require.ErrorIs(t, err, ErrValue)
	err = New(0).Make(nil)
	require.ErrorIs(t, err, ErrValue)

	_, err = Take(strings.NewReader("void main( {\n"))
	require.ErrorIs(t, err, ErrSyntax)
	require.ErrorIs(t, err, errs.ErrSyntax)

	_, err = Take(strings.NewReader("#include \"blur_common.h\"\n"))
	require.ErrorIs(t, err, ErrSyntax)

	_, err = Take(strings.NewReader(""))
	require.NoError(t, err)
}

func roundTrip(t *testing.T, name, src string) {
	t.Helper()
	first := mustTake(t, src)
	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf), name)
	made := buf.String()
	second, err := Take(strings.NewReader(made))
	require.NoError(t, err, "%s\n%s", name, made)
	require.Equal(t, first, second, "%s\n%s", name, made)
	var again bytes.Buffer
	require.NoError(t, second.Make(&again), name)
	require.Equal(t, made, again.String(), name)
}

func roundTripFile(t *testing.T, name string, file *File) {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf), name)
	made := buf.String()
	got, err := Take(strings.NewReader(made))
	require.NoError(t, err, "%s\n%s", name, made)
	require.Equal(t, file, got, "%s\n%s", name, made)
	roundTrip(t, name, made)
}

func compileShader(t *testing.T, name, src string, fallback Stage) {
	t.Helper()
	file := mustTake(t, src)
	stage := file.Stage
	if stage == 0 {
		stage = fallback
	}
	if stage == 0 {
		stage = stageFromName(name)
	}
	require.NotEqual(t, Stage(0), stage, "%s\n%s", name, src)
	if file.Stage != 0 {
		if fromName := stageFromName(name); fromName != 0 {
			require.Equal(t, fromName, file.Stage, name)
		}
	}
	made := mustMake(t, src)
	compileOne(t, name+" original", stage, src)
	compileOne(t, name+" reprint\n"+made, stage, made)
}

func compileOne(t *testing.T, name string, stage Stage, src string) {
	t.Helper()
	spv, err := glslang.CompileStage(t.Context(), ffiStage(stage), []byte(src))
	require.NoError(t, err, name)
	require.True(t, glslang.IsSPIRV(spv), name)
}

func ffiStage(stage Stage) glslang.Stage {
	switch stage {
	case Vertex:
		return glslang.StageVertex
	case Fragment:
		return glslang.StageFragment
	case Compute:
		return glslang.StageCompute
	default:
		return -1
	}
}

func stageFromName(name string) Stage {
	base := strings.ToLower(filepath.Base(name))
	switch {
	case strings.HasSuffix(base, ".vert") || strings.Contains(base, ".vertex."):
		return Vertex
	case strings.HasSuffix(base, ".frag") || strings.Contains(base, ".fragment."):
		return Fragment
	case strings.HasSuffix(base, ".comp") || strings.Contains(base, ".compute."):
		return Compute
	default:
		return 0
	}
}

func mustTake(t *testing.T, src string) *File {
	t.Helper()
	file, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	return file
}

func mustMake(t *testing.T, src string) string {
	t.Helper()
	file := mustTake(t, src)
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	return buf.String()
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func rawConst(t *testing.T, path, name string) string {
	t.Helper()
	text := readText(t, path)
	needle := "const " + name + " = `"
	i := strings.Index(text, needle)
	require.NotEqual(t, -1, i, name)
	rest := text[i+len(needle):]
	j := strings.Index(rest, "`")
	require.NotEqual(t, -1, j, name)
	return rest[:j]
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean(filepath.Join(packageDir(t), "..", "..", ".."))
}

func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Dir(file)
}
