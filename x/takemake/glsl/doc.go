// Package glsl is one GLSL shader.
//
// [New] returns an empty [File] for that stage. Methods return the
// receiver and store the child: version, uniform, input, output, and
// function. SetVersion and SetInstance avoid clashing with the fields
// of the same meaning. [Take] parses shader text. [Make] prints it.
// [Take], then [Make], then [Take] produce the same file. SPIR-V stays
// in x/ffi/wasm/glsl.
//
//	shader := glsl.New(glsl.Fragment).SetVersion(450)
//	shader.Uniform("color", glsl.Vec4())
//	shader.Func("main").Returns(glsl.Void()).Block()
//
// The contract is in DESIGN.md beside the parent directory.
package glsl
