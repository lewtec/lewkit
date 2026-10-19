package makefile

// Term is one makefile token. The scanner, not the grammar, decides when a
// tab is a recipe.
//
//lewkit:enum
type Term string

const (
	TermAssign   Term = "assign"
	TermRule     Term = "rule"
	TermRecipe   Term = "recipe"
	TermIf       Term = "if"
	TermElse     Term = "else"
	TermEndif    Term = "endif"
	TermDefine   Term = "define"
	TermBody     Term = "body"
	TermEndef    Term = "endef"
	TermInclude  Term = "include"
	TermVPath    Term = "vpath"
	TermExport   Term = "export"
	TermUndefine Term = "undefine"
	TermUnknown  Term = "unknown"
)

// Production is one makefile grammar rule. The AST rule is [Rule].
// Hidden rules the builder invents are not members.
//
//lewkit:enum
type Production string

const (
	ProductionFile      Production = "file"
	ProductionStatement Production = "statement"
	ProductionCondition Production = "condition"
	ProductionElse      Production = "else_arm"
	ProductionDefine    Production = "define_statement"
)
