// Package parse defines a language and parses it into one concrete tree.
//
// A [Grammar] is the walkable model: tokens, extras, and rules. [Grammar.Compile]
// turns that model into LR(1) tables. [Language.Parse] lexes and parses.
// The lexer returns the longest token among the terminals the current state
// accepts, so a keyword and a word that match the same bytes follow the state.
// A shift/reduce pair with no declared precedence stays in the table, and the
// parser forks a graph-structured stack (the fork shares its tail). Declared
// precedence and associativity drop the losing action while the tables are built.
//
// Nodes are records in one array. Children of a node are a contiguous range of
// indexes. Token text is a slice of the source the caller passed in; the tree
// does not copy it and does not allocate a string per node. A hidden rule
// (a name that starts with '_' or '$', including the names [Grammar.Repeat] and
// [Grammar.Opt] create) is flattened into its parent, which is the same idea as
// an inlined tree-sitter rule.
//
// A language whose tokens depend on earlier tokens, such as Go's inserted
// semicolon, sets [Grammar.Scanner]. That scanner reads the file once.
// The default lexer is stateless and state-directed.
//
// Parse fails with an error that unwraps to [errs.ErrSyntax]. There is no
// error-recovery node in this version: a fork that cannot move is dropped, and
// the furthest such point is the error.
package parse
