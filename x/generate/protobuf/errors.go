package protobuf

import "errors"

var (
	errFileRequired = errors.New("proto file required")
	errPlugin       = errors.New("protoc-gen-go")
	errProtoc       = errors.New("protoc")
)
