// Package sarif is one SARIF 2.1.0 log.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: run, result, rule, and fix. [Take] parses the log. [Make]
// prints it. x/text/report keeps findings and its SARIF printer. The
// contract is in DESIGN.md beside the parent directory. This scaffold
// returns [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package sarif
