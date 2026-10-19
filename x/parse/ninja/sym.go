package ninja

// Term is one ninja token. Keyword texts are the words ninja 1.12 reserves
// at the start of a line. Value and path are modes of that lexer, not a
// second spelling of an identifier.
//
//lewkit:enum
type Term string

const (
	TermIdentifier Term = "identifier"
	TermBuild      Term = "build"
	TermPool       Term = "pool"
	TermRule       Term = "rule"
	TermDefault    Term = "default"
	TermInclude    Term = "include"
	TermSubninja   Term = "subninja"
	TermEqual      Term = "="
	TermColon      Term = ":"
	TermPipe       Term = "|"
	TermDoublePipe Term = "||"
	TermPipeAtSign Term = "|@"
	TermIndent     Term = "indent"
	TermNewline    Term = "newline"
	TermValue      Term = "value"
	TermPath       Term = "path"
)

// Production is one ninja grammar rule. Hidden rules the builder invents are not
// members. The name is not Rule: that type is the build rule in the file model.
// Spellings stay off the token texts, since a keyword and a rule cannot share one.
//
//lewkit:enum
type Production string

const (
	ProductionFile     Production = "file"
	ProductionVariable Production = "variable"
	ProductionRule     Production = "rule_declaration"
	ProductionPool     Production = "pool_declaration"
	ProductionEdge     Production = "edge"
	ProductionDefault  Production = "default_declaration"
	ProductionInclude  Production = "include_declaration"
	ProductionSubninja Production = "subninja_declaration"
	ProductionBinding  Production = "binding"
	ProductionOutputs  Production = "outputs"
	ProductionInputs   Production = "inputs"
)
