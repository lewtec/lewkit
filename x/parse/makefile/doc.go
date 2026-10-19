// Package makefile parses one Makefile into the shared concrete tree.
//
// The declaration has no scanner. x/takemake/makefile installs the scanner
// that classifies logical lines, then loads the generated table.
package makefile
