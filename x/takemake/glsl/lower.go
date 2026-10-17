package glsl

import (
	"errors"
	"fmt"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

func parseSource(src string) (*File, error) {
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse([]byte(src))
	if err != nil {
		var pe *xparse.Error
		if errors.As(err, &pe) {
			return nil, fmt.Errorf("%w: line %d", ErrSyntax, pe.Line)
		}
		return nil, fmt.Errorf("%w: line 1", ErrSyntax)
	}
	return lowerFile(src, root)
}

func lowerFile(src string, root xparse.Syn[Term, Rule]) (*File, error) {
	file := New(0)
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RulePrecisionStatement:
			item, err := lowerPrecision(src, child)
			if err != nil {
				return nil, err
			}
			file.Precisions = append(file.Precisions, item)
		case RuleStructDeclaration:
			item, decls, err := lowerStruct(src, child)
			if err != nil {
				return nil, err
			}
			file.Structs = append(file.Structs, item)
			file.Globals = append(file.Globals, decls...)
		case RuleInterface:
			decl, err := lowerIface(src, child)
			if err != nil {
				return nil, err
			}
			file.Globals = append(file.Globals, decl)
		case RuleFunc:
			fn, err := lowerFunc(src, child)
			if err != nil {
				return nil, err
			}
			file.Functions = append(file.Functions, fn)
		case RuleDeclaration:
			decls, err := lowerDecl(src, child)
			if err != nil {
				return nil, err
			}
			file.Globals = append(file.Globals, decls...)
		case RuleNamed:
			decl, err := lowerNamed(src, child)
			if err != nil {
				return nil, err
			}
			file.Globals = append(file.Globals, decl)
		case RuleBare:
			decl, err := lowerBare(src, child)
			if err != nil {
				return nil, err
			}
			file.Globals = append(file.Globals, decl)
		default:
			return nil, syntaxf(src, child.Start(), "declaration")
		}
	}
	return file, nil
}

func lowerPrecision(src string, node xparse.Syn[Term, Rule]) (Precision, error) {
	var item Precision
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if ok && rule == RuleRank {
			rank, err := ParseRank(textOf(child))
			if err != nil {
				return Precision{}, syntaxf(src, child.Start(), "precision %s", textOf(child))
			}
			item.Rank = rank
			continue
		}
		if term, ok := child.Term(); ok && term == TermIdentifier {
			item.Type = child.Text()
		}
	}
	if !item.Rank.IsValid() || item.Type == "" {
		return Precision{}, syntaxf(src, node.Start(), "precision")
	}
	return item, nil
}

func lowerStruct(src string, node xparse.Syn[Term, Rule]) (*Struct, []*Decl, error) {
	item := &Struct{Name: identText(node)}
	if item.Name == "" {
		return nil, nil, syntaxf(src, node.Start(), "struct")
	}
	var extra []*Decl
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleDeclaration:
			fields, err := lowerDecl(src, child)
			if err != nil {
				return nil, nil, err
			}
			item.Fields = append(item.Fields, fields...)
		case RuleTail:
			decls, err := lowerTail(src, child, item.Name)
			if err != nil {
				return nil, nil, err
			}
			extra = append(extra, decls...)
		}
	}
	return item, extra, nil
}

func lowerTail(src string, node xparse.Syn[Term, Rule], typeName string) ([]*Decl, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleDeclarators {
			return lowerDeclarators(src, child, qualState{}, Type{Name: typeName})
		}
	}
	return nil, nil
}

func lowerIface(src string, node xparse.Syn[Term, Rule]) (*Decl, error) {
	qual, err := lowerQuals(src, node)
	if err != nil {
		return nil, err
	}
	decl := qual.decl()
	decl.Block = identText(node)
	if decl.Block == "" {
		return nil, syntaxf(src, node.Start(), "block")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleDeclaration:
			fields, err := lowerDecl(src, child)
			if err != nil {
				return nil, err
			}
			decl.Fields = append(decl.Fields, fields...)
		case RuleInstance:
			decl.Instance = identText(child)
			sizes, err := sizesOf(src, child)
			if err != nil {
				return nil, err
			}
			decl.Sizes = sizes
		}
	}
	return decl, nil
}

func lowerFunc(src string, node xparse.Syn[Term, Rule]) (*Func, error) {
	qual, err := lowerQuals(src, node)
	if err != nil {
		return nil, err
	}
	if qual.storage != 0 || qual.invariant || qual.precise || qual.interp != 0 || qual.aux != 0 || len(qual.layouts) > 0 || len(qual.memory) > 0 {
		return nil, syntaxf(src, node.Start(), "function qualifier")
	}
	fn := &Func{Rank: qual.rank}
	var names int
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier {
			if names == 0 {
				fn.Return.Name = child.Text()
			} else if names == 1 {
				fn.Name = child.Text()
			}
			names++
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleParameterList:
			params, err := lowerParams(src, child)
			if err != nil {
				return nil, err
			}
			fn.Params = params
		case RuleBracket:
			size, err := lowerBracket(src, child)
			if err != nil {
				return nil, err
			}
			fn.Return.Sizes = append(fn.Return.Sizes, size)
		case RuleBody:
			body, err := lowerBlockStmts(src, child)
			if err != nil {
				return nil, err
			}
			fn.Body = body
		}
	}
	if fn.Return.Name == "" || fn.Name == "" {
		return nil, syntaxf(src, node.Start(), "function")
	}
	if len(fn.Return.Sizes) == 0 {
		fn.Return.Sizes = nil
	}
	return fn, nil
}

func lowerParams(src string, node xparse.Syn[Term, Rule]) ([]*Param, error) {
	var params []*Param
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleParameter {
			continue
		}
		param, err := lowerParam(src, child)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	if len(params) == 1 && bareVoid(params[0]) {
		return nil, nil
	}
	return params, nil
}

func bareVoid(param *Param) bool {
	return param != nil && !param.Const && param.Storage == 0 && param.Rank == 0 && len(param.Memory) == 0 &&
		param.Name == "" && param.Type.Name == "void" && len(param.Type.Sizes) == 0
}

func lowerParam(src string, node xparse.Syn[Term, Rule]) (*Param, error) {
	param := &Param{}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			if term, ok := child.Term(); ok && term == TermIdentifier {
				param.Type.Name = child.Text()
			}
			continue
		}
		switch rule {
		case RuleParameterQualifier:
			if err := applyPQual(src, param, child); err != nil {
				return nil, err
			}
		case RuleParameterName:
			name, sizes, err := lowerPName(src, child)
			if err != nil {
				return nil, err
			}
			param.Name = name
			param.Type.Sizes = sizes
		}
	}
	if param.Type.Name == "" {
		return nil, syntaxf(src, node.Start(), "parameter")
	}
	return param, nil
}

func applyPQual(src string, param *Param, node xparse.Syn[Term, Rule]) error {
	term, text := keyword(node)
	switch term {
	case TermConst:
		param.Const = true
	case TermIn, TermOut, TermInOut:
		storage, err := ParseStorage(text)
		if err != nil {
			return syntaxf(src, node.Start(), "parameter %s", text)
		}
		param.Storage = storage
	case TermLowp, TermMediump, TermHighp:
		rank, err := ParseRank(text)
		if err != nil {
			return syntaxf(src, node.Start(), "parameter %s", text)
		}
		param.Rank = rank
	case TermCoherent, TermVolatile, TermRestrict, TermReadonly, TermWriteonly:
		mem, err := ParseMemory(text)
		if err != nil {
			return syntaxf(src, node.Start(), "parameter %s", text)
		}
		param.Memory = addMemory(param.Memory, mem)
	default:
		return syntaxf(src, node.Start(), "parameter")
	}
	return nil
}

func lowerPName(src string, node xparse.Syn[Term, Rule]) (string, []Size, error) {
	name := identText(node)
	sizes, err := sizesOf(src, node)
	if err != nil {
		return "", nil, err
	}
	return name, sizes, nil
}

func lowerNamed(src string, node xparse.Syn[Term, Rule]) (*Decl, error) {
	qual, err := lowerQuals(src, node)
	if err != nil {
		return nil, err
	}
	decl := qual.decl()
	decl.Name = identText(node)
	if decl.Name == "" {
		return nil, syntaxf(src, node.Start(), "name")
	}
	return decl, nil
}

func lowerBare(src string, node xparse.Syn[Term, Rule]) (*Decl, error) {
	qual, err := lowerQuals(src, node)
	if err != nil {
		return nil, err
	}
	return qual.decl(), nil
}

func lowerDecl(src string, node xparse.Syn[Term, Rule]) ([]*Decl, error) {
	qual, err := lowerQuals(src, node)
	if err != nil {
		return nil, err
	}
	typeName := identText(node)
	if typeName == "" {
		return nil, syntaxf(src, node.Start(), "declaration")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleDeclarators {
			return lowerDeclarators(src, child, qual, Type{Name: typeName})
		}
	}
	return nil, syntaxf(src, node.Start(), "declaration")
}

func lowerDeclarators(src string, node xparse.Syn[Term, Rule], qual qualState, typ Type) ([]*Decl, error) {
	var decls []*Decl
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleDeclarator {
			continue
		}
		decl, err := lowerDeclarator(src, child, qual, typ)
		if err != nil {
			return nil, err
		}
		decls = append(decls, decl)
	}
	if len(decls) == 0 {
		return nil, syntaxf(src, node.Start(), "declarator")
	}
	return decls, nil
}

func lowerDeclarator(src string, node xparse.Syn[Term, Rule], qual qualState, typ Type) (*Decl, error) {
	decl := qual.decl()
	decl.Name = identText(node)
	if decl.Name == "" {
		return nil, syntaxf(src, node.Start(), "declarator")
	}
	sizes, err := sizesOf(src, node)
	if err != nil {
		return nil, err
	}
	decl.Type = withSizes(typ, sizes)
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleInitializer {
			expr, err := lowerInit(src, child)
			if err != nil {
				return nil, err
			}
			decl.Init = expr
		}
	}
	return decl, nil
}

func lowerInit(src string, node xparse.Syn[Term, Rule]) (Expr, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleAssign, RuleExpression:
			return lowerExpr(src, child)
		case RuleList:
			return lowerList(src, child)
		}
	}
	return nil, syntaxf(src, node.Start(), "initializer")
}

func lowerQuals(src string, node xparse.Syn[Term, Rule]) (qualState, error) {
	var qual qualState
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleQualifier {
			continue
		}
		if err := applyQual(src, &qual, child); err != nil {
			return qualState{}, err
		}
	}
	return qual, nil
}

func applyQual(src string, qual *qualState, node xparse.Syn[Term, Rule]) error {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleLayoutQualifier {
			items, err := lowerLayouts(src, child)
			if err != nil {
				return err
			}
			qual.layouts = append(qual.layouts, items...)
			continue
		}
		term, text := keyword(child)
		switch term {
		case TermInvariant:
			qual.invariant = true
		case TermPrecise:
			qual.precise = true
		case TermSmooth, TermFlat, TermNoPerspective:
			interp, err := ParseInterp(text)
			if err != nil || (qual.interp != 0 && qual.interp != interp) {
				return syntaxf(src, child.Start(), "interpolation %s", text)
			}
			qual.interp = interp
		case TermCentroid, TermSample, TermPatch:
			aux, err := ParseAux(text)
			if err != nil || (qual.aux != 0 && qual.aux != aux) {
				return syntaxf(src, child.Start(), "auxiliary %s", text)
			}
			qual.aux = aux
		case TermCoherent, TermVolatile, TermRestrict, TermReadonly, TermWriteonly:
			mem, err := ParseMemory(text)
			if err != nil {
				return syntaxf(src, child.Start(), "memory %s", text)
			}
			qual.memory = addMemory(qual.memory, mem)
		case TermLowp, TermMediump, TermHighp:
			rank, err := ParseRank(text)
			if err != nil || (qual.rank != 0 && qual.rank != rank) {
				return syntaxf(src, child.Start(), "precision %s", text)
			}
			qual.rank = rank
		case TermConst, TermIn, TermOut, TermInOut, TermUniform, TermBuffer, TermShared:
			storage, err := ParseStorage(text)
			if err != nil || (qual.storage != 0 && qual.storage != storage) {
				return syntaxf(src, child.Start(), "storage %s", text)
			}
			qual.storage = storage
		default:
			if term != "" {
				return syntaxf(src, child.Start(), "qualifier %s", text)
			}
		}
	}
	return nil
}

func lowerLayouts(src string, node xparse.Syn[Term, Rule]) ([]Layout, error) {
	var items []Layout
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleLayoutItems {
			continue
		}
		var current *Layout
		for j := 0; j < child.ChildCount(); j++ {
			part := child.Child(j)
			if term, ok := part.Term(); ok && term == TermIdentifier {
				items = append(items, Layout{Name: part.Text()})
				current = &items[len(items)-1]
				continue
			}
			if rule, ok := part.Rule(); ok && (rule == RuleAssign || rule == RuleExpression) {
				if current == nil {
					return nil, syntaxf(src, part.Start(), "layout")
				}
				expr, err := lowerExpr(src, part)
				if err != nil {
					return nil, err
				}
				current.Value = expr
			}
		}
	}
	return items, nil
}

type qualState struct {
	layouts   []Layout
	invariant bool
	precise   bool
	interp    Interp
	aux       Aux
	memory    []Memory
	rank      Rank
	storage   Storage
}

func (qual qualState) decl() *Decl {
	decl := &Decl{
		Invariant: qual.invariant,
		Precise:   qual.precise,
		Interp:    qual.interp,
		Aux:       qual.aux,
		Rank:      qual.rank,
		Storage:   qual.storage,
	}
	if len(qual.layouts) > 0 {
		decl.Layouts = append([]Layout{}, qual.layouts...)
	}
	if len(qual.memory) > 0 {
		decl.Memory = append([]Memory{}, qual.memory...)
	}
	return decl
}

func lowerStmt(src string, node xparse.Syn[Term, Rule]) (Stmt, error) {
	node = unwrap(node, RuleStatement, RuleSwitchItem)
	rule, ok := node.Rule()
	if !ok {
		return nil, syntaxf(src, node.Start(), "statement")
	}
	switch rule {
	case RuleEmpty:
		return &Empty{}, nil
	case RuleBlock:
		body, err := lowerBlockStmts(src, node)
		if err != nil {
			return nil, err
		}
		return &Block{Body: body}, nil
	case RuleIfStatement:
		return lowerIf(src, node)
	case RuleForStatement:
		return lowerFor(src, node)
	case RuleWhileStatement:
		return lowerWhile(src, node)
	case RuleDoStatement:
		return lowerDo(src, node)
	case RuleSwitchStatement:
		return lowerSwitch(src, node)
	case RuleReturnStatement:
		return lowerReturn(src, node)
	case RuleBreakStatement:
		return &Break{}, nil
	case RuleContinueStatement:
		return &Continue{}, nil
	case RuleDiscardStatement:
		return &Discard{}, nil
	case RuleDeclaration:
		decls, err := lowerDecl(src, node)
		if err != nil {
			return nil, err
		}
		if len(decls) == 1 {
			return decls[0], nil
		}
		body := make([]Stmt, len(decls))
		for i, decl := range decls {
			body[i] = decl
		}
		return &Block{Body: body}, nil
	case RuleExpressionStatement:
		expr, err := firstExpr(src, node)
		if err != nil {
			return nil, err
		}
		return &ExprStmt{Expr: expr}, nil
	default:
		return nil, syntaxf(src, node.Start(), "statement")
	}
}

func lowerBlockStmts(src string, node xparse.Syn[Term, Rule]) ([]Stmt, error) {
	var body []Stmt
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleStatement {
			continue
		}
		stmt, err := lowerStmt(src, child)
		if err != nil {
			return nil, err
		}
		if stmt != nil {
			body = append(body, stmt)
		}
	}
	if body == nil {
		return []Stmt{}, nil
	}
	return body, nil
}

func lowerControlled(src string, node xparse.Syn[Term, Rule]) ([]Stmt, error) {
	node = unwrap(node, RuleStatement)
	if rule, ok := node.Rule(); ok && rule == RuleBlock {
		return lowerBlockStmts(src, node)
	}
	stmt, err := lowerStmt(src, node)
	if err != nil {
		return nil, err
	}
	if stmt == nil {
		return []Stmt{}, nil
	}
	return []Stmt{stmt}, nil
}

func lowerIf(src string, node xparse.Syn[Term, Rule]) (*If, error) {
	item := &If{}
	var stmts []xparse.Syn[Term, Rule]
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		if rule == RuleExpression && item.Cond == nil {
			expr, err := lowerExpr(src, child)
			if err != nil {
				return nil, err
			}
			item.Cond = expr
			continue
		}
		if rule == RuleStatement {
			stmts = append(stmts, child)
		}
	}
	if item.Cond == nil || len(stmts) == 0 {
		return nil, syntaxf(src, node.Start(), "if")
	}
	body, err := lowerControlled(src, stmts[0])
	if err != nil {
		return nil, err
	}
	item.Body = body
	if len(stmts) > 1 {
		els, err := lowerControlled(src, stmts[1])
		if err != nil {
			return nil, err
		}
		item.Else = els
	}
	return item, nil
}

func lowerFor(src string, node xparse.Syn[Term, Rule]) (*For, error) {
	inner, bodyNode, err := parenInner(node)
	if err != nil {
		return nil, syntaxf(src, node.Start(), "for")
	}
	item := &For{}
	if len(inner) == 0 {
		return nil, syntaxf(src, node.Start(), "for")
	}
	rest := inner
	switch {
	case isText(rest[0], ";"):
		rest = rest[1:]
	case isRule(rest[0], RuleDeclaration):
		init, err := lowerStmt(src, rest[0])
		if err != nil {
			return nil, err
		}
		item.Init = init
		rest = rest[1:]
	case isRule(rest[0], RuleExpression):
		expr, err := lowerExpr(src, rest[0])
		if err != nil {
			return nil, err
		}
		item.Init = &ExprStmt{Expr: expr}
		if len(rest) == 0 || !isText(rest[1], ";") {
			return nil, syntaxf(src, node.Start(), "for")
		}
		rest = rest[2:]
	default:
		return nil, syntaxf(src, rest[0].Start(), "for")
	}
	cond, post, err := splitForRest(src, rest)
	if err != nil {
		return nil, err
	}
	item.Cond = cond
	item.Post = post
	body, err := lowerControlled(src, bodyNode)
	if err != nil {
		return nil, err
	}
	item.Body = body
	return item, nil
}

func splitForRest(src string, nodes []xparse.Syn[Term, Rule]) (Expr, Expr, error) {
	var left, right []xparse.Syn[Term, Rule]
	seen := false
	for _, node := range nodes {
		if !seen && isText(node, ";") {
			seen = true
			continue
		}
		if !seen {
			left = append(left, node)
			continue
		}
		right = append(right, node)
	}
	if !seen {
		if len(nodes) == 0 {
			return nil, nil, syntaxf(src, 0, "for")
		}
		return nil, nil, syntaxf(src, nodes[0].Start(), "for")
	}
	cond, err := optionalExpr(src, left)
	if err != nil {
		return nil, nil, err
	}
	post, err := optionalExpr(src, right)
	if err != nil {
		return nil, nil, err
	}
	return cond, post, nil
}

func optionalExpr(src string, nodes []xparse.Syn[Term, Rule]) (Expr, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	if len(nodes) != 1 {
		return nil, syntaxf(src, nodes[0].Start(), "expression")
	}
	return lowerExpr(src, nodes[0])
}

func lowerWhile(src string, node xparse.Syn[Term, Rule]) (*While, error) {
	expr, err := firstExpr(src, node)
	if err != nil {
		return nil, err
	}
	bodyNode := lastStmt(node)
	if !bodyNode.Valid() {
		return nil, syntaxf(src, node.Start(), "while")
	}
	body, err := lowerControlled(src, bodyNode)
	if err != nil {
		return nil, err
	}
	return &While{Cond: expr, Body: body}, nil
}

func lowerDo(src string, node xparse.Syn[Term, Rule]) (*Do, error) {
	bodyNode := xparse.Syn[Term, Rule]{}
	var exprNode xparse.Syn[Term, Rule]
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		if rule == RuleStatement && !bodyNode.Valid() {
			bodyNode = child
			continue
		}
		if (rule == RuleExpression || rule == RuleAssign) && !exprNode.Valid() {
			exprNode = child
		}
	}
	if !bodyNode.Valid() || !exprNode.Valid() {
		return nil, syntaxf(src, node.Start(), "do")
	}
	body, err := lowerControlled(src, bodyNode)
	if err != nil {
		return nil, err
	}
	expr, err := lowerExpr(src, exprNode)
	if err != nil {
		return nil, err
	}
	return &Do{Body: body, Cond: expr}, nil
}

func lowerSwitch(src string, node xparse.Syn[Term, Rule]) (*Switch, error) {
	expr, err := firstExpr(src, node)
	if err != nil {
		return nil, err
	}
	item := &Switch{Expr: expr}
	var current *Case
	for i := 0; i < node.ChildCount(); i++ {
		child := unwrap(node.Child(i), RuleSwitchItem)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleCaseLabel:
			label, err := firstExpr(src, child)
			if err != nil {
				return nil, err
			}
			current = &Case{Expr: label}
			item.Cases = append(item.Cases, current)
		case RuleDefaultLabel:
			current = &Case{Default: true}
			item.Cases = append(item.Cases, current)
		case RuleStatement:
			if current == nil {
				return nil, syntaxf(src, child.Start(), "case")
			}
			stmt, err := lowerStmt(src, child)
			if err != nil {
				return nil, err
			}
			current.Body = append(current.Body, stmt)
		}
	}
	return item, nil
}

func lowerReturn(src string, node xparse.Syn[Term, Rule]) (*Return, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && (rule == RuleExpression || rule == RuleAssign) {
			expr, err := lowerExpr(src, child)
			if err != nil {
				return nil, err
			}
			return &Return{Expr: expr}, nil
		}
	}
	return &Return{}, nil
}

func lowerExpr(src string, node xparse.Syn[Term, Rule]) (Expr, error) {
	if !node.Valid() {
		return nil, syntaxf(src, 0, "expression")
	}
	if _, ok := node.Rule(); !ok {
		return lowerAtom(src, node)
	}
	switch node.ChildCount() {
	case 1:
		return lowerExpr(src, node.Child(0))
	case 2:
		left, right := node.Child(0), node.Child(1)
		if _, ok := left.Term(); ok {
			expr, err := lowerExpr(src, right)
			if err != nil {
				return nil, err
			}
			return &Unary{Op: left.Text(), Expr: expr}, nil
		}
		expr, err := lowerExpr(src, left)
		if err != nil {
			return nil, err
		}
		return &Postfix{Expr: expr, Op: right.Text()}, nil
	case 3:
		first, mid, last := node.Child(0), node.Child(1), node.Child(2)
		if isText(first, "(") {
			return lowerExpr(src, mid)
		}
		if isText(mid, ".") {
			expr, err := lowerExpr(src, first)
			if err != nil {
				return nil, err
			}
			return &Select{Expr: expr, Name: last.Text()}, nil
		}
		if isText(mid, "(") && isText(last, ")") {
			expr, err := lowerExpr(src, first)
			if err != nil {
				return nil, err
			}
			return &Call{Callee: expr}, nil
		}
		return lowerBinary(src, first, mid, last)
	case 4:
		first, mid, third, last := node.Child(0), node.Child(1), node.Child(2), node.Child(3)
		expr, err := lowerExpr(src, first)
		if err != nil {
			return nil, err
		}
		if isText(mid, "[") {
			index, err := lowerExpr(src, third)
			if err != nil {
				return nil, err
			}
			return &Index{Base: expr, Index: index}, nil
		}
		if isText(mid, "(") {
			args, err := lowerArgs(src, third)
			if err != nil {
				return nil, err
			}
			_ = last
			return &Call{Callee: expr, Args: args}, nil
		}
	case 5:
		cond, err := lowerExpr(src, node.Child(0))
		if err != nil {
			return nil, err
		}
		then, err := lowerExpr(src, node.Child(2))
		if err != nil {
			return nil, err
		}
		els, err := lowerExpr(src, node.Child(4))
		if err != nil {
			return nil, err
		}
		return &Ternary{Cond: cond, Then: then, Else: els}, nil
	}
	return nil, syntaxf(src, node.Start(), "expression")
}

func lowerBinary(src string, left, op, right xparse.Syn[Term, Rule]) (Expr, error) {
	lhs, err := lowerExpr(src, left)
	if err != nil {
		return nil, err
	}
	rhs, err := lowerExpr(src, right)
	if err != nil {
		return nil, err
	}
	return &Binary{Op: op.Text(), Left: lhs, Right: rhs}, nil
}

func lowerArgs(src string, node xparse.Syn[Term, Rule]) ([]Expr, error) {
	var args []Expr
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok || (rule != RuleAssign && rule != RuleExpression) {
			continue
		}
		expr, err := lowerExpr(src, child)
		if err != nil {
			return nil, err
		}
		args = append(args, expr)
	}
	return args, nil
}

func lowerList(src string, node xparse.Syn[Term, Rule]) (*List, error) {
	var elems []Expr
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleElements {
			if err := collectElems(src, child, &elems); err != nil {
				return nil, err
			}
		}
	}
	return &List{Exprs: elems}, nil
}

func collectElems(src string, node xparse.Syn[Term, Rule], out *[]Expr) error {
	if node.ChildCount() == 1 {
		return appendElem(src, node.Child(0), out)
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleElements:
			if err := collectElems(src, child, out); err != nil {
				return err
			}
		case RuleElement, RuleAssign, RuleExpression, RuleList:
			if err := appendElem(src, child, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func appendElem(src string, node xparse.Syn[Term, Rule], out *[]Expr) error {
	node = unwrap(node, RuleElement)
	rule, ok := node.Rule()
	if !ok {
		return syntaxf(src, node.Start(), "initializer")
	}
	var expr Expr
	var err error
	switch rule {
	case RuleList:
		expr, err = lowerList(src, node)
	default:
		expr, err = lowerExpr(src, node)
	}
	if err != nil {
		return err
	}
	*out = append(*out, expr)
	return nil
}

func lowerAtom(src string, node xparse.Syn[Term, Rule]) (Expr, error) {
	term, ok := node.Term()
	if !ok {
		return nil, syntaxf(src, node.Start(), "expression")
	}
	switch term {
	case TermIdentifier:
		switch node.Text() {
		case "true":
			return &Literal{Kind: KindBool, Bool: true}, nil
		case "false":
			return &Literal{Kind: KindBool, Bool: false}, nil
		default:
			return &Ident{Name: node.Text()}, nil
		}
	case TermNumber:
		lit, end, err := scanNumber(node.Text(), 0, lineAt(src, node.Start()))
		if err != nil {
			return nil, err
		}
		if end != len(node.Text()) {
			return nil, syntaxf(src, node.Start(), "number")
		}
		return &lit, nil
	default:
		return nil, syntaxf(src, node.Start(), "expression")
	}
}

func firstExpr(src string, node xparse.Syn[Term, Rule]) (Expr, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && (rule == RuleExpression || rule == RuleAssign) {
			return lowerExpr(src, child)
		}
	}
	return nil, syntaxf(src, node.Start(), "expression")
}

func lastStmt(node xparse.Syn[Term, Rule]) xparse.Syn[Term, Rule] {
	var stmt xparse.Syn[Term, Rule]
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleStatement {
			stmt = child
		}
	}
	return stmt
}

func parenInner(node xparse.Syn[Term, Rule]) ([]xparse.Syn[Term, Rule], xparse.Syn[Term, Rule], error) {
	var inner []xparse.Syn[Term, Rule]
	var stmt xparse.Syn[Term, Rule]
	saw := false
	done := false
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if !saw {
			if isText(child, "(") {
				saw = true
			}
			continue
		}
		if !done && isText(child, ")") {
			done = true
			continue
		}
		if done {
			stmt = child
			continue
		}
		inner = append(inner, child)
	}
	if !saw || !done || !stmt.Valid() {
		return nil, xparse.Syn[Term, Rule]{}, fmt.Errorf("%w: for", ErrSyntax)
	}
	return inner, stmt, nil
}

func sizesOf(src string, node xparse.Syn[Term, Rule]) ([]Size, error) {
	var sizes []Size
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleBracket {
			continue
		}
		size, err := lowerBracket(src, child)
		if err != nil {
			return nil, err
		}
		sizes = append(sizes, size)
	}
	return sizes, nil
}

func lowerBracket(src string, node xparse.Syn[Term, Rule]) (Size, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && (rule == RuleAssign || rule == RuleExpression) {
			expr, err := lowerExpr(src, child)
			if err != nil {
				return Size{}, err
			}
			return Size{Expr: expr}, nil
		}
	}
	return Size{}, nil
}

func withSizes(typ Type, extra []Size) Type {
	if len(extra) == 0 {
		if len(typ.Sizes) == 0 {
			return typ
		}
		typ.Sizes = append([]Size{}, typ.Sizes...)
		return typ
	}
	sizes := make([]Size, 0, len(typ.Sizes)+len(extra))
	sizes = append(sizes, typ.Sizes...)
	sizes = append(sizes, extra...)
	typ.Sizes = sizes
	return typ
}

func identText(node xparse.Syn[Term, Rule]) string {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier {
			return child.Text()
		}
	}
	return ""
}

func textOf(node xparse.Syn[Term, Rule]) string {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if _, ok := child.Term(); ok {
			return child.Text()
		}
	}
	return ""
}

func keyword(node xparse.Syn[Term, Rule]) (Term, string) {
	node = unwrap(node, RuleQualifier, RuleParameterQualifier, RuleRank)
	term, ok := node.Term()
	if !ok {
		return "", ""
	}
	return term, node.Text()
}

func unwrap(node xparse.Syn[Term, Rule], rules ...Rule) xparse.Syn[Term, Rule] {
	for node.Valid() && node.ChildCount() == 1 {
		rule, ok := node.Rule()
		if !ok || !ruleOne(rule, rules) {
			return node
		}
		node = node.Child(0)
	}
	return node
}

func ruleOne(rule Rule, rules []Rule) bool {
	for _, item := range rules {
		if rule == item {
			return true
		}
	}
	return false
}

func isRule(node xparse.Syn[Term, Rule], rule Rule) bool {
	got, ok := node.Rule()
	return ok && got == rule
}

func isText(node xparse.Syn[Term, Rule], text string) bool {
	return node.Valid() && node.Text() == text
}

func lineAt(src string, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	return strings.Count(src[:off], "\n") + 1
}

func syntaxf(src string, off int, format string, args ...any) error {
	return fmt.Errorf("%w: line %d: %s", ErrSyntax, lineAt(src, off), fmt.Sprintf(format, args...))
}
