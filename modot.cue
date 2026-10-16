package modot

inputs: capstone_wasm: {
	from:    "github:wasilibs/go-capstone"
	version: "HEAD"
}
inputs: onnx: {
	from:    "github:onnx/onnx"
	version: "v1.22.0"
}
inputs: htmx: {
	from:    "github:bigskysoftware/htmx"
	version: "v4.0.0"
}
inputs: jquery: {
	from:    "github:jquery/jquery"
	version: "4.0.0"
}
inputs: sakuracss: {
	from:    "github:oxalorg/sakura"
	version: "1.5.1"
}
inputs: tailwindcss: {
	from:    "github:tailwindlabs/tailwindcss"
	version: "v4.3.3"
}
inputs: golang: {
	from:    "github:golang/go"
	version: "go1.27.0"
}
// Vulkan-Samples publishes no release tags. main is the pin Renovate tracks.
inputs: vulkan_samples: {
	from:    "github:KhronosGroup/Vulkan-Samples"
	version: "main"
}
inputs: lua: {
	from:    "github:lua/lua"
	version: "v5.4.7"
}
inputs: zlib: {
	from:    "github:madler/zlib"
	version: "v1.3.1"
}
inputs: ninja: {
	from:    "github:ninja-build/ninja"
	version: "v1.12.1"
}
inputs: graphviz: {
	from:    "github:ellson/MOTHBALLED-graphviz"
	version: "master"
}

lazy_tools: protobuf: {
	ref:  "github:protocolbuffers/protobuf"
	bins: ["protoc"]
}
lazy_tools: sops: {
	ref:  "github:getsops/sops"
	bins: ["sops"]
}

// The browser bundle is not in the git tag. Place that package.json
// as upstream.package.json. tailwindcss.js stays in the repo.
file: codebase: {
	"x/ffi/wasm/capstone/internal/wasm": {
		type: "place"
		source: "capstone_wasm:internal/wasm/libcapstone.wasm"
		steps: "10_require": {
			op: "require"
			patterns: wasm: "libcapstone.wasm"
		}
	}
	"x/ndarray/onnx/internal/proto": {
		type: "place"
		source: "onnx:onnx/onnx.proto"
		steps: "10_require": {
			op: "require"
			patterns: proto: "onnx.proto"
		}
	}
	"x/http/asset/htmx": {
		type: "place"
		source: "htmx:dist/htmx.min.js"
		steps: "10_require": {
			op: "require"
			patterns: js: "htmx.min.js"
		}
	}
	"x/http/asset/jquery": {
		type: "place"
		source: "jquery:dist/jquery.min.js"
		steps: "10_require": {
			op: "require"
			patterns: js: "jquery.min.js"
		}
	}
	"x/http/asset/sakuracss": {
		type: "place"
		source: "sakuracss:css/sakura.css"
		steps: "10_require": {
			op: "require"
			patterns: css: "sakura.css"
		}
	}
	"x/http/asset/tailwindcss": {
		type: "place"
		source: "tailwindcss:packages/@tailwindcss-browser/package.json"
		steps: {
			"10_require": {
				op: "require"
				patterns: json: "package.json"
			}
			"20_rename": {
				op:   "move"
				from: "package.json"
				to:   "upstream.package.json"
			}
		}
	}
	// GLSL fixtures. A file source with no steps lands on the dest path, so
	// the names x/takemake/glsl tests walk stay flat. shaders/triangle.frag
	// is the ES sample. shaders/deferred/lighting.vert keeps its sample name.
	// blur_down.comp still has #include, which Take rejects.
	"x/takemake/glsl/testdata/khronos/base.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/dynamic_uniform_buffers/glsl/base.frag"
	}
	"x/takemake/glsl/testdata/khronos/binary_demo.comp": {
		type:   "place"
		source: "vulkan_samples:shaders/pipeline_binary/glsl/binary_demo.comp"
	}
	"x/takemake/glsl/testdata/khronos/blur_down.comp": {
		type:   "place"
		source: "vulkan_samples:shaders/async_compute/blur_down.comp"
	}
	"x/takemake/glsl/testdata/khronos/composition.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/hdr/glsl/composition.frag"
	}
	"x/takemake/glsl/testdata/khronos/composition.vert": {
		type:   "place"
		source: "vulkan_samples:shaders/hdr/glsl/composition.vert"
	}
	"x/takemake/glsl/testdata/khronos/cube.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/push_descriptors/glsl/cube.frag"
	}
	"x/takemake/glsl/testdata/khronos/deferred_lighting.vert": {
		type:   "place"
		source: "vulkan_samples:shaders/deferred/lighting.vert"
	}
	"x/takemake/glsl/testdata/khronos/game_of_life_update.comp": {
		type:   "place"
		source: "vulkan_samples:shaders/timeline_semaphore/glsl/game_of_life_update.comp"
	}
	"x/takemake/glsl/testdata/khronos/imgui.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/imgui.frag"
	}
	"x/takemake/glsl/testdata/khronos/particle_calculate.comp": {
		type:   "place"
		source: "vulkan_samples:shaders/compute_nbody/glsl/particle_calculate.comp"
	}
	"x/takemake/glsl/testdata/khronos/particle_integrate.comp": {
		type:   "place"
		source: "vulkan_samples:shaders/compute_nbody/glsl/particle_integrate.comp"
	}
	"x/takemake/glsl/testdata/khronos/postprocessing.vert": {
		type:   "place"
		source: "vulkan_samples:shaders/postprocessing/postprocessing.vert"
	}
	"x/takemake/glsl/testdata/khronos/shadowmap.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/shadows/shadowmap.frag"
	}
	"x/takemake/glsl/testdata/khronos/shadowmap.vert": {
		type:   "place"
		source: "vulkan_samples:shaders/shadows/shadowmap.vert"
	}
	"x/takemake/glsl/testdata/khronos/triangle.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/hello_triangle/glsl/triangle.frag"
	}
	"x/takemake/glsl/testdata/khronos/triangle.vert": {
		type:   "place"
		source: "vulkan_samples:shaders/hello_triangle/glsl/triangle.vert"
	}
	"x/takemake/glsl/testdata/khronos/triangle_es.frag": {
		type:   "place"
		source: "vulkan_samples:shaders/triangle.frag"
	}
	// Pristine Go 1.27 compiler, linker, and the std internal packages they
	// import. A directory named _upstream is invisible to the go tool.
	// x/toolchain/go/rewrite turns this pin into packages the module can import.
	"x/toolchain/go/_upstream/compile": {
		type:   "place"
		source: "golang:src/cmd/compile"
	}
	"x/toolchain/go/_upstream/link": {
		type:   "place"
		source: "golang:src/cmd/link"
	}
	"x/toolchain/go/_upstream/cmd": {
		type:   "place"
		source: "golang:src/cmd/internal"
	}
	"x/toolchain/go/_upstream/std": {
		type:   "place"
		source: "golang:src/internal"
	}
	"x/toolchain/go/_upstream/telemetry": {
		type:   "place"
		source: "golang:src/cmd/vendor/golang.org/x/telemetry"
	}
	// Lua 5.4.7 makefile and zlib 1.3.1 Makefile.in.
	// A file source with no steps lands on the dest path itself.
	"x/takemake/makefile/testdata/makefile": {
		type:   "place"
		source: "lua:makefile"
	}
	"x/takemake/makefile/testdata/Makefile.in": {
		type:   "place"
		source: "zlib:Makefile.in"
	}
	// Ninja 1.12.1 build files that ship in the repository.
	"x/takemake/ninja/testdata": {
		type: "place"
		source: {
			slow: "ninja:misc/long-slow-build.ninja"
			fds:  "ninja:misc/inherited-fds.ninja"
			afl:  "ninja:misc/afl-fuzz/build.ninja"
		}
		steps: "10_require": {
			op: "require"
			patterns: ninja: "*.ninja"
		}
	}
	// Graphviz examples (EPL-2.0) from the mothballed GitHub tree.
	"x/takemake/graphviz/testdata": {
		type: "place"
		source: {
			abstract: "graphviz:graphs/directed/abstract.gv"
			arrows:   "graphviz:graphs/directed/arrows.gv"
			clust:    "graphviz:graphs/directed/clust.gv"
			clust1:   "graphviz:graphs/directed/clust1.gv"
			dfa:      "graphviz:graphs/directed/dfa.gv"
			fsm:      "graphviz:graphs/directed/fsm.gv"
			grammar:  "graphviz:graphs/directed/grammar.gv"
			nan:      "graphviz:graphs/directed/NaN.gv"
			records:  "graphviz:graphs/directed/records.gv"
			structs:  "graphviz:graphs/directed/structs.gv"
			switch:   "graphviz:graphs/directed/switch.gv"
			table:    "graphviz:graphs/directed/table.gv"
			unix:     "graphviz:graphs/directed/unix.gv"
			world:    "graphviz:graphs/directed/world.gv"
			er:       "graphviz:graphs/undirected/ER.gv"
			petersen: "graphviz:graphs/undirected/Petersen.gv"
		}
		steps: "10_require": {
			op: "require"
			patterns: gv: "*.gv"
		}
	}
}

// The prelude formatter is `gofmt -w .`. That walks the Go pin, and the
// pin's testdata does not parse. Format the rest of the module instead.
formatter: tools: gofmt: enable: false
formatter: tools: "go-sources": {
	detect: "00-go-mod": {
		path:   "go.mod"
		enable: true
	}
	cmd: ["sh", "x/toolchain/format.sh"]
}

// Prelude golangci-lint and govulncheck run on ./... . That includes
// x/toolchain/go/src, the rewritten compiler. Point them at the same
// package list as the tests.
lint: tools: "golangci-lint": enable: false
lint: tools: govulncheck: enable: false
lint: tools: "go-lint": {
	detect: "00-go-mod": {
		path:   "go.mod"
		enable: true
	}
	needs: golangci_lint: true
	cmd: ["sh", "x/toolchain/lint-golangci.sh"]
	output: "sarif"
}
lint: tools: "go-vuln": {
	detect: "00-go-mod": {
		path:   "go.mod"
		enable: true
	}
	needs: govulncheck: true
	cmd: ["sh", "x/toolchain/lint-govuln.sh"]
	output: "sarif"
}
