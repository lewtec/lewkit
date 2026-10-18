package sql

import (
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse

// Language loads the SQL grammar.
func Language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		prepared, err := declare()
		if err != nil {
			langErr = err
			return
		}
		lang, langErr = prepared.Load(sqlTable)
	})
	return lang, langErr
}

func declare() (*xparse.Prepared[Term, Rule], error) {
	var syntax xparse.Syntax[Term, Rule]

	identifier := syntax.Word(TermIdentifier, identifierStart, identifierContinue)
	stringLiteral := syntax.Pat(TermString, xparse.Func(MatchString))
	numberLiteral := syntax.Pat(TermNumber, xparse.Func(MatchNumber))
	place := syntax.Pat(TermPlaceholder, xparse.Func(MatchPlaceholder))
	quoted := syntax.Pat(TermQuoted, xparse.Func(MatchQuoted))
	alias := syntax.Pat(TermAlias, xparse.Func(matchAlias))
	space := syntax.Skip(syntax.Pat(TermSpace, xparse.While(" \t\n\r")))
	lineNote := syntax.Skip(syntax.Pat(TermLineComment, xparse.Func(matchLineComment)))
	blockNote := syntax.Skip(syntax.Pat(TermBlockComment, xparse.Func(matchBlockComment)))

	keyword := func(kind Term) *xparse.Piece[Term, Rule] {
		return syntax.Lex(2, syntax.Term(kind, matchKeyword(kind.String())))
	}
	action := keyword(TermAction)
	add := keyword(TermAdd)
	alter := keyword(TermAlter)
	and := keyword(TermAnd)
	as := keyword(TermAs)
	asc := keyword(TermAsc)
	auto := keyword(TermAutoincrement)
	by := keyword(TermBy)
	cascade := keyword(TermCascade)
	check := keyword(TermCheck)
	column := keyword(TermColumn)
	conflict := keyword(TermConflict)
	constraint := keyword(TermConstraint)
	create := keyword(TermCreate)
	data := keyword(TermData)
	defaultKeyword := keyword(TermDefault)
	deleteKeyword := keyword(TermDelete)
	desc := keyword(TermDesc)
	distinct := keyword(TermDistinct)
	do := keyword(TermDo)
	drop := keyword(TermDrop)
	exists := keyword(TermExists)
	foreign := keyword(TermForeign)
	from := keyword(TermFrom)
	group := keyword(TermGroup)
	ifKeyword := keyword(TermIf)
	index := keyword(TermIndex)
	insert := keyword(TermInsert)
	into := keyword(TermInto)
	is := keyword(TermIs)
	key := keyword(TermKey)
	like := keyword(TermLike)
	limit := keyword(TermLimit)
	no := keyword(TermNo)
	not := keyword(TermNot)
	nothing := keyword(TermNothing)
	null := keyword(TermNull)
	on := keyword(TermOn)
	or := keyword(TermOr)
	order := keyword(TermOrder)
	primaryKeyword := keyword(TermPrimary)
	references := keyword(TermReferences)
	rename := keyword(TermRename)
	restrict := keyword(TermRestrict)
	selectKeyword := keyword(TermSelect)
	set := keyword(TermSet)
	table := keyword(TermTable)
	to := keyword(TermTo)
	typeWord := keyword(TermTypeWord)
	unique := keyword(TermUnique)
	update := keyword(TermUpdate)
	values := keyword(TermValues)
	where := keyword(TermWhere)

	leftParenthesis := syntax.Lit("(")
	rightParenthesis := syntax.Lit(")")
	comma := syntax.Lit(",")
	semicolon := syntax.Lit(";")
	dot := syntax.Lit(".")
	star := syntax.Lit("*")
	equal := syntax.Lit("=")
	less := syntax.Lit("<")
	greater := syntax.Lit(">")
	plus := syntax.Lit("+")
	minus := syntax.Lit("-")
	slash := syntax.Lit("/")
	percent := syntax.Lit("%")
	notEqual := syntax.Lit("<>")
	bang := syntax.Lit("!=")
	lessEqual := syntax.Lit("<=")
	greaterEqual := syntax.Lit(">=")
	concat := syntax.Lit("||")

	name := syntax.Eq(RuleName, syntax.Alt(identifier, quoted))
	qualifiedName := syntax.Eq(RuleQualifiedName, syntax.Sep1(syntax.Ref(RuleName), dot))
	typeName := syntax.Eq(RuleTypeName, identifier)
	parenNames := syntax.Eq(RuleParenthesizedNames, syntax.Seq(leftParenthesis, syntax.Sep1(syntax.Ref(RuleName), comma), rightParenthesis))
	ifNot := syntax.Eq(RuleIfNotExists, syntax.Seq(ifKeyword, not, exists))
	ifYes := syntax.Eq(RuleIfExists, syntax.Seq(ifKeyword, exists))

	// Layers follow the old recursive descent. NOT's operand includes
	// comparisons, and a comparison's operand does not include NOT.
	expr := syntax.Eq(RuleExpression, syntax.Ref(RuleOr))
	orExpr := syntax.Eq(RuleOr, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleOr), or, syntax.Ref(RuleAnd)),
		syntax.Ref(RuleAnd),
	))
	andExpr := syntax.Eq(RuleAnd, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleAnd), and, syntax.Ref(RuleNot)),
		syntax.Ref(RuleNot),
	))
	notExpr := syntax.Eq(RuleNot, syntax.Alt(
		syntax.Seq(not, syntax.Ref(RuleNot)),
		syntax.Ref(RuleComparison),
	))
	cmpExpr := syntax.Eq(RuleComparison, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleComparison), equal, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), notEqual, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), bang, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), lessEqual, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), greaterEqual, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), less, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), greater, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), like, syntax.Ref(RuleSum)),
		syntax.Seq(syntax.Ref(RuleComparison), is, not, null),
		syntax.Seq(syntax.Ref(RuleComparison), is, null),
		syntax.Ref(RuleSum),
	))
	sumExpr := syntax.Eq(RuleSum, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleSum), plus, syntax.Ref(RuleProduct)),
		syntax.Seq(syntax.Ref(RuleSum), minus, syntax.Ref(RuleProduct)),
		syntax.Seq(syntax.Ref(RuleSum), concat, syntax.Ref(RuleProduct)),
		syntax.Ref(RuleProduct),
	))
	productExpr := syntax.Eq(RuleProduct, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleProduct), star, syntax.Ref(RuleUnary)),
		syntax.Seq(syntax.Ref(RuleProduct), slash, syntax.Ref(RuleUnary)),
		syntax.Seq(syntax.Ref(RuleProduct), percent, syntax.Ref(RuleUnary)),
		syntax.Ref(RuleUnary),
	))
	unaryExpr := syntax.Eq(RuleUnary, syntax.Alt(
		syntax.Seq(plus, syntax.Ref(RuleUnary)),
		syntax.Seq(minus, syntax.Ref(RuleUnary)),
		syntax.Ref(RulePrimary),
	))
	// Prec keeps the call's '('. A bare name would otherwise reduce.
	call := syntax.Eq(RuleCall, syntax.Prec(2, syntax.Seq(
		syntax.Ref(RuleQualifiedName), leftParenthesis, syntax.Sep(syntax.Ref(RuleExpression), comma), rightParenthesis,
	)))
	primary := syntax.Eq(RulePrimary, syntax.Alt(
		syntax.Ref(RuleCall),
		syntax.Ref(RuleQualifiedName),
		stringLiteral,
		numberLiteral,
		place,
		star,
		syntax.Seq(leftParenthesis, syntax.Ref(RuleExpression), rightParenthesis),
	))

	act := syntax.Eq(RuleAction, syntax.Alt(
		cascade,
		restrict,
		syntax.Seq(set, null),
		syntax.Seq(set, defaultKeyword),
		syntax.Seq(no, action),
	))
	onAction := syntax.Eq(RuleOnAction, syntax.Seq(on, syntax.Alt(deleteKeyword, update), syntax.Ref(RuleAction)))
	colConstraint := syntax.Eq(RuleColumnConstraint, syntax.Alt(
		syntax.Seq(constraint, syntax.Ref(RuleName)),
		syntax.Seq(primaryKeyword, key, syntax.Opt(auto), syntax.Opt(asc), syntax.Opt(desc)),
		syntax.Seq(not, null),
		null,
		unique,
		auto,
		syntax.Seq(defaultKeyword, syntax.Ref(RuleUnary)),
		syntax.Seq(references, syntax.Ref(RuleName), syntax.Ref(RuleParenthesizedNames)),
		syntax.Ref(RuleOnAction),
	))
	col := syntax.Eq(RuleColumn, syntax.Seq(
		syntax.Ref(RuleName), syntax.Ref(RuleTypeName), syntax.Repeat(syntax.Ref(RuleColumnConstraint)),
	))
	tableKey := syntax.Eq(RuleTableKey, syntax.Seq(primaryKeyword, key, syntax.Ref(RuleParenthesizedNames)))
	tableUnique := syntax.Eq(RuleTableUnique, syntax.Seq(unique, syntax.Ref(RuleParenthesizedNames)))
	tableForeign := syntax.Eq(RuleTableForeign, syntax.Seq(
		foreign, key, syntax.Ref(RuleParenthesizedNames),
		references, syntax.Ref(RuleName), syntax.Ref(RuleParenthesizedNames),
		syntax.Repeat(syntax.Ref(RuleOnAction)),
	))
	tableCheck := syntax.Eq(RuleTableCheck, syntax.Seq(check, leftParenthesis, syntax.Ref(RuleExpression), rightParenthesis))
	tableConstraint := syntax.Eq(RuleTableConstraint, syntax.Seq(
		syntax.Opt(syntax.Seq(constraint, syntax.Ref(RuleName))),
		syntax.Alt(
			syntax.Ref(RuleTableKey),
			syntax.Ref(RuleTableUnique),
			syntax.Ref(RuleTableForeign),
			syntax.Ref(RuleTableCheck),
		),
	))
	tableItem := syntax.Eq(RuleTableItem, syntax.Alt(
		syntax.Ref(RuleTableConstraint),
		syntax.Ref(RuleColumn),
	))

	createTable := syntax.Eq(RuleCreateTable, syntax.Seq(
		create, table, syntax.Opt(syntax.Ref(RuleIfNotExists)), syntax.Ref(RuleName),
		leftParenthesis, syntax.Sep(syntax.Ref(RuleTableItem), comma), rightParenthesis,
	))
	createIndex := syntax.Eq(RuleCreateIndex, syntax.Seq(
		create, syntax.Opt(unique), index, syntax.Opt(syntax.Ref(RuleIfNotExists)),
		syntax.Opt(syntax.Ref(RuleName)), on, syntax.Ref(RuleName), syntax.Ref(RuleParenthesizedNames),
		syntax.Opt(syntax.Ref(RuleWhere)),
	))

	alterRename := syntax.Eq(RuleAlterRename, syntax.Seq(rename, to, syntax.Ref(RuleName)))
	alterRenameCol := syntax.Eq(RuleAlterRenameColumn, syntax.Seq(
		rename, syntax.Opt(column), syntax.Ref(RuleName), to, syntax.Ref(RuleName),
	))
	alterAdd := syntax.Eq(RuleAlterAdd, syntax.Seq(add, syntax.Opt(column), syntax.Ref(RuleColumn)))
	alterDrop := syntax.Eq(RuleAlterDrop, syntax.Seq(drop, syntax.Opt(column), syntax.Ref(RuleName)))
	alterType := syntax.Eq(RuleAlterType, syntax.Seq(
		alter, syntax.Opt(column), syntax.Ref(RuleName),
		syntax.Opt(syntax.Seq(set, data)), typeWord, syntax.Ref(RuleTypeName),
	))
	alterAction := syntax.Eq(RuleAlterAction, syntax.Alt(
		syntax.Ref(RuleAlterRename),
		syntax.Ref(RuleAlterRenameColumn),
		syntax.Ref(RuleAlterAdd),
		syntax.Ref(RuleAlterDrop),
		syntax.Ref(RuleAlterType),
	))
	alterStmt := syntax.Eq(RuleAlter, syntax.Seq(
		alter, table, syntax.Ref(RuleName), syntax.Ref(RuleAlterAction),
	))
	dropStmt := syntax.Eq(RuleDrop, syntax.Seq(
		drop, table, syntax.Opt(syntax.Ref(RuleIfExists)), syntax.Ref(RuleName),
		syntax.Opt(syntax.Alt(cascade, restrict)),
	))

	selectItem := syntax.Eq(RuleSelectItem, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleExpression), as, syntax.Ref(RuleName)),
		syntax.Seq(syntax.Ref(RuleExpression), alias),
		syntax.Ref(RuleExpression),
	))
	fromClause := syntax.Eq(RuleFrom, syntax.Seq(from, syntax.Sep1(syntax.Ref(RuleName), comma)))
	whereClause := syntax.Eq(RuleWhere, syntax.Seq(where, syntax.Ref(RuleExpression)))
	groupClause := syntax.Eq(RuleGroup, syntax.Seq(group, by, syntax.Sep1(syntax.Ref(RuleExpression), comma)))
	orderItem := syntax.Eq(RuleOrderItem, syntax.Alt(
		syntax.Seq(syntax.Ref(RuleExpression), desc),
		syntax.Seq(syntax.Ref(RuleExpression), asc),
		syntax.Ref(RuleExpression),
	))
	orderClause := syntax.Eq(RuleOrder, syntax.Seq(order, by, syntax.Sep1(syntax.Ref(RuleOrderItem), comma)))
	limitClause := syntax.Eq(RuleLimit, syntax.Seq(limit, syntax.Ref(RuleExpression)))
	selectStmt := syntax.Eq(RuleSelect, syntax.Seq(
		selectKeyword, syntax.Opt(distinct), syntax.Sep1(syntax.Ref(RuleSelectItem), comma),
		syntax.Opt(syntax.Ref(RuleFrom)), syntax.Opt(syntax.Ref(RuleWhere)),
		syntax.Opt(syntax.Ref(RuleGroup)), syntax.Opt(syntax.Ref(RuleOrder)),
		syntax.Opt(syntax.Ref(RuleLimit)),
	))

	row := syntax.Eq(RuleRow, syntax.Seq(leftParenthesis, syntax.Sep1(syntax.Ref(RuleExpression), comma), rightParenthesis))
	assign := syntax.Eq(RuleAssign, syntax.Seq(syntax.Ref(RuleName), equal, syntax.Ref(RuleExpression)))
	conflictClause := syntax.Eq(RuleConflict, syntax.Seq(
		on, conflict, syntax.Opt(syntax.Ref(RuleParenthesizedNames)), do,
		syntax.Alt(nothing, syntax.Seq(update, set, syntax.Sep1(syntax.Ref(RuleAssign), comma))),
	))
	insertStmt := syntax.Eq(RuleInsert, syntax.Seq(
		insert, into, syntax.Ref(RuleName), syntax.Opt(syntax.Ref(RuleParenthesizedNames)),
		values, syntax.Sep1(syntax.Ref(RuleRow), comma), syntax.Opt(syntax.Ref(RuleConflict)),
	))
	updateStmt := syntax.Eq(RuleUpdate, syntax.Seq(
		update, syntax.Ref(RuleName), set, syntax.Sep1(syntax.Ref(RuleAssign), comma),
		syntax.Opt(syntax.Ref(RuleWhere)),
	))
	deleteStmt := syntax.Eq(RuleDelete, syntax.Seq(
		deleteKeyword, from, syntax.Ref(RuleName), syntax.Opt(syntax.Ref(RuleWhere)),
	))

	stmt := syntax.Eq(RuleStatement, syntax.Alt(
		syntax.Ref(RuleCreateTable),
		syntax.Ref(RuleCreateIndex),
		syntax.Ref(RuleAlter),
		syntax.Ref(RuleDrop),
		syntax.Ref(RuleSelect),
		syntax.Ref(RuleInsert),
		syntax.Ref(RuleUpdate),
		syntax.Ref(RuleDelete),
	))
	file := syntax.Eq(RuleFile, syntax.Repeat(syntax.Alt(syntax.Ref(RuleStatement), semicolon)))

	return syntax.Prepare("sql", syntax.Compose(
		file, stmt,
		createTable, createIndex, alterStmt, dropStmt,
		selectStmt, insertStmt, updateStmt, deleteStmt,
		ifNot, ifYes, name, qualifiedName, typeName, parenNames,
		col, colConstraint, tableItem, tableConstraint,
		tableKey, tableUnique, tableForeign, tableCheck,
		onAction, act,
		expr, orExpr, andExpr, notExpr, cmpExpr, sumExpr, productExpr, unaryExpr, primary, call,
		selectItem, fromClause, whereClause, groupClause, orderItem, orderClause, limitClause,
		row, assign, conflictClause,
		alterRename, alterRenameCol, alterAdd, alterDrop, alterType, alterAction,
		space, lineNote, blockNote,
		syntax.Start(RuleFile),
	))
}
