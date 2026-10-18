package makefile

import grammar "github.com/lewtec/lewkit/x/parse/makefile"

// Term and Production are the makefile tokens and rules. The grammar lives in x/parse/makefile.
type (
	Term       = grammar.Term
	Production = grammar.Production
)

const (
	TermAssign          = grammar.TermAssign
	TermRule            = grammar.TermRule
	TermRecipe          = grammar.TermRecipe
	TermIf              = grammar.TermIf
	TermElse            = grammar.TermElse
	TermEndif           = grammar.TermEndif
	TermDefine          = grammar.TermDefine
	TermBody            = grammar.TermBody
	TermEndef           = grammar.TermEndef
	TermInclude         = grammar.TermInclude
	TermVPath           = grammar.TermVPath
	TermExport          = grammar.TermExport
	TermUndefine        = grammar.TermUndefine
	TermUnknown         = grammar.TermUnknown
	ProductionFile      = grammar.ProductionFile
	ProductionStatement = grammar.ProductionStatement
	ProductionCondition = grammar.ProductionCondition
	ProductionElse      = grammar.ProductionElse
	ProductionDefine    = grammar.ProductionDefine
)
