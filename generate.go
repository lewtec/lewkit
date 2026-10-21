package generate

//go:generate go tool modot codebase apply
//go:generate go run ./x/http/asset/lucide/genicons.go
//go:generate go run ./x/http/asset/lucide/genversion.go
//go:generate go run ./x/toolchain/go/rewrite
//go:generate go run ./cmd/lewkit generate prelude
