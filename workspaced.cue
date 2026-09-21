package workspaced

workspaced: {
	runtime: inputs: capstone_wasm: path: string | *""
	inputs: capstone_wasm: {
		from:    "github:wasilibs/go-capstone"
		version: "HEAD"
	}
	modules: capstone_wasm: {
		from: "core:place"
		config: {
			items: {
				"x/ffi/wasm/capstone/internal/wasm": "capstone_wasm:internal/wasm/libcapstone.wasm"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						wasm: "libcapstone.wasm"
					}
				}
			}
		}
	}
	file: codebase: "x/ffi/wasm/capstone/internal/wasm/libcapstone.wasm": {
		type: "ref"
		values: {
			src: {
				kind: "ref"
				ref:  "\(workspaced.runtime.inputs.capstone_wasm.path)/internal/wasm/libcapstone.wasm"
			}
		}
	}
}

workspaced: {
	runtime: inputs: glslang_wasm: path: string | *""
	// generate.sh (mise emscripten/cmake/ninja, uv python) writes glslang.wasm.
	modules: glslang_wasm: {
		from: "core:place"
		config: {
			items: {
				"x/ffi/wasm/glsl/internal/wasm": "glslang_wasm:glslang.wasm"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						wasm: "glslang.wasm"
					}
				}
			}
		}
	}
}

workspaced: {
	lazy_tools: {
		protobuf: {
			ref:  "github:protocolbuffers/protobuf"
			bins: ["protoc"]
		}
	}
}

workspaced: {
	runtime: inputs: onnx: path: string | *""
	inputs: onnx: {
		from:    "github:onnx/onnx"
		version: "v1.22.0"
	}
	modules: onnx_proto: {
		from: "core:place"
		config: {
			items: {
				"x/ndarray/onnx/internal/proto": "onnx:onnx/onnx.proto"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						proto: "onnx.proto"
					}
				}
			}
		}
	}
	file: codebase: "x/ndarray/onnx/internal/proto/onnx.proto": {
		type: "ref"
		values: {
			src: {
				kind: "ref"
				ref:  "\(workspaced.runtime.inputs.onnx.path)/onnx/onnx.proto"
			}
		}
	}
}
