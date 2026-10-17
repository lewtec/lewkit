// Package pe is one PE image.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: machine, section, import, and export. [Take] reads a PE
// into that file. [Make] writes it. The contract is in DESIGN.md beside
// the parent directory.
package pe
