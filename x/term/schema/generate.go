// Package schema is the protobuf schema for the term socket.
// Each socket message is one varint size-delimited Envelope.
package schema

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate protobuf term.proto
