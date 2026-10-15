// Package glsl is one GLSL shader.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: stage, version, uniform, and function. [Take] and [Make]
// are the shader text. SPIR-V stays in x/ffi/wasm/glsl. The contract is
// in DESIGN.md beside the parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package glsl
