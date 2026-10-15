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
// .modot/config renders x/http/asset/lucide/version_gen.go from this version.
inputs: lucide: {
	from:    "github:lucide-icons/lucide"
	version: "1.57.0"
}
inputs: golang: {
	from:    "github:golang/go"
	version: "go1.27.0"
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
