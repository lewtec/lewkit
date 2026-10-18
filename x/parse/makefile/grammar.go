package makefile

import xparse "github.com/lewtec/lewkit/x/parse"

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse

// Declare places the makefile grammar. Take installs the scanner before Load.
// The scanner is not in the table.
func Declare() (*xparse.Prepared[Term, Production], error) {
	return declare()
}

// Table is the LR(1) automaton generated from Declare.
func Table() []byte { return makefileTable }

func declare() (*xparse.Prepared[Term, Production], error) {
	var syntax xparse.Syntax[Term, Production]
	assign := syntax.Term(TermAssign, nil)
	rule := syntax.Term(TermRule, nil)
	recipe := syntax.Term(TermRecipe, nil)
	ifTok := syntax.Term(TermIf, nil)
	elseTok := syntax.Term(TermElse, nil)
	endif := syntax.Term(TermEndif, nil)
	defineTok := syntax.Term(TermDefine, nil)
	body := syntax.Term(TermBody, nil)
	endef := syntax.Term(TermEndef, nil)
	include := syntax.Term(TermInclude, nil)
	vpath := syntax.Term(TermVPath, nil)
	export := syntax.Term(TermExport, nil)
	undefine := syntax.Term(TermUndefine, nil)
	unknown := syntax.Term(TermUnknown, nil)

	statement := syntax.Eq(ProductionStatement, syntax.Alt(
		assign,
		rule,
		recipe,
		include,
		vpath,
		export,
		undefine,
		unknown,
		syntax.Ref(ProductionCondition),
		syntax.Ref(ProductionDefine),
	))
	elseBranch := syntax.Eq(ProductionElse, syntax.Seq(
		elseTok,
		syntax.Repeat(syntax.Ref(ProductionStatement)),
	))
	condition := syntax.Eq(ProductionCondition, syntax.Seq(
		ifTok,
		syntax.Repeat(syntax.Ref(ProductionStatement)),
		syntax.Repeat(syntax.Ref(ProductionElse)),
		endif,
	))
	// endef is optional so a missing closer is reported on the define line.
	defineRule := syntax.Eq(ProductionDefine, syntax.Seq(
		defineTok,
		syntax.Repeat(body),
		syntax.Opt(endef),
	))
	file := syntax.Eq(ProductionFile, syntax.Repeat(syntax.Ref(ProductionStatement)))
	return syntax.Prepare("makefile", syntax.Compose(
		file,
		statement,
		elseBranch,
		condition,
		defineRule,
		syntax.Start(ProductionFile),
	))
}
