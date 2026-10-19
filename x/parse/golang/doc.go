// Package golang parses one Go source file into the shared concrete tree.
//
// The scanner is the Go scanner, including inserted semicolons, so the token
// stream matches go/scanner. The grammar is this package's rules compiled by
// x/parse. Parse does not call go/parser.
package golang
