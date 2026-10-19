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
//
// A [Form] is a grammar expression. [Eq] binds a name. [Seq], [Alt], [Repeat],
// and [Opt] combine expressions. [Compose] unions them: two equations with one
// name become alternatives of that rule. [Build] places the expression into a
// [Grammar]. The same form can be built into more than one grammar. [Left],
// [Right], and [Prec] wrap an alternative. [Lex] sets the lexical
// precedence of a token.
//
// A [Syntax] carries a token enum and a rule enum and stores nothing. Its
// methods return a [Piece]. [Syntax.Compile] places that piece and builds the
// tables. [Model.Parse] returns a [SyntaxNode]. [SyntaxNode.Term] and
// [SyntaxNode.Rule] read the enum value of a visible symbol. A hidden rule
// is neither. [Syntax.Lit] is
// punctuation with no enum value.
//
// [Syntax.Pat] binds a token to a [Pattern]: [While], [Literal], [Keyword],
// [Class], [Line], or [Quote]. [Pattern.Unless] and [After] consult the
// previous token. [Syntax.Flag] and [Syntax.Text] name fields that
// [SyntaxNode.Fill] copies. [Piece.Build] is the function for a rule whose
// value is not that copy. [Modes] and [Drive] are the scanner memory.
// [Syntax.Prepare] places the declaration. [Prepared.Load] reads the
// LR(1) table written by lewkit generate parse.
//
// [FromTreeSitter] loads a tree-sitter grammar.json. token rules become
// matchers, extras are skipped, and inline rules are substituted into their
// parents. An external-scanner token is a terminal with no matcher. An
// immediate token matches only when no trivia was just skipped.
package parse
