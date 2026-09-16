package workspaced

workspaced: {
	runtime: inputs: capstone_wasm: path: string | *""
	inputs: {
		capstone_wasm: {
			from:    "github:wasilibs/go-capstone"
			version: "HEAD"
		}
	}
	modules: {
		capstone_wasm: {
			from: "core:place"
			config: {
				items: {
					"x/disasm/internal/wasm": "capstone_wasm:internal/wasm/libcapstone.wasm"
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
	}
	file: {
		codebase: {
			"x/disasm/internal/wasm/libcapstone.wasm": {
				type: "ref"
				values: {
					src: {
						kind: "ref"
						ref:  "\(workspaced.runtime.inputs.capstone_wasm.path)/internal/wasm/libcapstone.wasm"
					}
				}
			}
		}
	}
}
