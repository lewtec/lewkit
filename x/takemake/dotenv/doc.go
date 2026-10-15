// Package dotenv is one assignment list.
//
// [New] returns an empty [File]. Methods return the receiver and store
// an assignment in order, including a comment and the original quoting.
// [Take] parses the file. [Make] prints it. x/sops.ParseEnv reads a
// secret file and drops comments. The contract is in DESIGN.md beside
// the parent directory. This scaffold returns
// [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package dotenv
