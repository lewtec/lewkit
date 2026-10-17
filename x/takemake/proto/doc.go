// Package proto is one Protocol Buffer file.
//
// [NewFile] returns an empty [File]. Methods return the receiver and
// store the child: syntax, package, import, option, message, enum, and
// service. A field is a name, a number, a type, and a label. [Take]
// parses proto text with an x/parse grammar and lowers the concrete
// tree. [Make] prints it. [Take], then [Make], then [Take] produce the
// same file.
//
// [Make] prints a canonical file. Inside a message the order is options,
// reserves, enums, nested messages, fields, then oneofs. [Take] groups
// the same way, so the round trip matches those fields.
//
// The contract is in DESIGN.md beside the parent directory.
package proto
