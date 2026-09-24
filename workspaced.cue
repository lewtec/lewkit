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

workspaced: {
	runtime: inputs: htmx: path: string | *""
	inputs: htmx: {
		from:    "github:bigskysoftware/htmx"
		version: "v4.0.0"
	}
	modules: htmx: {
		from: "core:place"
		config: {
			items: {
				"x/http/asset/htmx": "htmx:dist/htmx.min.js"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						js: "htmx.min.js"
					}
				}
			}
		}
	}
	file: codebase: "x/http/asset/htmx/htmx.min.js": {
		type: "ref"
		values: {
			src: {
				kind: "ref"
				ref:  "\(workspaced.runtime.inputs.htmx.path)/dist/htmx.min.js"
			}
		}
	}
}

workspaced: {
	runtime: inputs: jquery: path: string | *""
	inputs: jquery: {
		from:    "github:jquery/jquery"
		version: "4.0.0"
	}
	modules: jquery: {
		from: "core:place"
		config: {
			items: {
				"x/http/asset/jquery": "jquery:dist/jquery.min.js"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						js: "jquery.min.js"
					}
				}
			}
		}
	}
	file: codebase: "x/http/asset/jquery/jquery.min.js": {
		type: "ref"
		values: {
			src: {
				kind: "ref"
				ref:  "\(workspaced.runtime.inputs.jquery.path)/dist/jquery.min.js"
			}
		}
	}
}

workspaced: {
	runtime: inputs: sakuracss: path: string | *""
	inputs: sakuracss: {
		from:    "github:oxalorg/sakura"
		version: "1.5.1"
	}
	modules: sakuracss: {
		from: "core:place"
		config: {
			items: {
				"x/http/asset/sakuracss": "sakuracss:css/sakura.css"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						css: "sakura.css"
					}
				}
			}
		}
	}
	file: codebase: "x/http/asset/sakuracss/sakura.css": {
		type: "ref"
		values: {
			src: {
				kind: "ref"
				ref:  "\(workspaced.runtime.inputs.sakuracss.path)/css/sakura.css"
			}
		}
	}
}

// The browser bundle is not in the git tag. It is the npm package
// @tailwindcss/browser at the same version, copied to tailwindcss.js.
workspaced: {
	runtime: inputs: tailwindcss: path: string | *""
	inputs: tailwindcss: {
		from:    "github:tailwindlabs/tailwindcss"
		version: "v4.3.3"
	}
	modules: tailwindcss: {
		from: "core:place"
		config: {
			items: {
				"x/http/asset/tailwindcss": "tailwindcss:packages/@tailwindcss-browser/package.json"
			}
			steps: {
				"10_require": {
					op: "require"
					patterns: {
						json: "package.json"
					}
				}
			}
		}
	}
	file: codebase: "x/http/asset/tailwindcss/upstream.package.json": {
		type: "ref"
		values: {
			src: {
				kind: "ref"
				ref:  "\(workspaced.runtime.inputs.tailwindcss.path)/packages/@tailwindcss-browser/package.json"
			}
		}
	}
}
