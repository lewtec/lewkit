// Package c is one C translation unit.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: include, type, const, func, and object. A function keeps
// its body. A header is that file with no bodies. Size, alignment, and
// field offset use the LP64 System V layout and live on the type.
// [Take] parses C. [Make] prints C.
package c
