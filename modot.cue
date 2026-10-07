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
}
