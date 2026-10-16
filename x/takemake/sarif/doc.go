// Package sarif is one SARIF 2.1.0 log.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: run, result, rule, and fix. [Take] parses the log. [Make]
// prints it. [Take], then [Make], then [Take] produce the same file.
//
// v1 reads runs, driver rules, results, physical locations, and fixes.
// A message is plain text and optional markdown. A property outside that
// set, or a version other than 2.1.0, is rejected. x/text/report keeps
// findings and its SARIF printer.
//
// The contract is in DESIGN.md beside the parent directory.
package sarif
