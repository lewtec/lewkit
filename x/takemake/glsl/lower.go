package glsl

import (
	"errors"
	"fmt"

	xparse "github.com/lewtec/lewkit/x/parse"
	syntax "github.com/lewtec/lewkit/x/parse/glsl"
)

func parseSource(src string) (*File, error) {
	model, err := syntax.Language()
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

func lowerFile(src string, root xparse.SyntaxNode[Term, Rule]) (*File, error) {
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
			return nil, syntaxf(child, "declaration")
		}
	}
	return file, nil
}

func lowerPrecision(src string, node xparse.SyntaxNode[Term, Rule]) (Precision, error) {
	var item Precision
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if ok && rule == RuleRank {
			rank, err := ParseRank(textOf(child))
			if err != nil {
				return Precision{}, syntaxf(child, "precision %s", textOf(child))
			}
			item.Rank = rank
			continue
		}
		if term, ok := child.Term(); ok && term == TermIdentifier {
			item.Type = child.Text()
		}
	}
	if !item.Rank.IsValid() || item.Type == "" {
		return Precision{}, syntaxf(node, "precision")
	}
	return item, nil
}

func lowerStruct(src string, node xparse.SyntaxNode[Term, Rule]) (*Struct, []*Decl, error) {
	item := &Struct{Name: identText(node)}
	if item.Name == "" {
		return nil, nil, syntaxf(node, "struct")
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

func lowerTail(src string, node xparse.SyntaxNode[Term, Rule], typeName string) ([]*Decl, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleDeclarators {
			return lowerDeclarators(src, child, qualifierState{}, Type{Name: typeName})
		}
	}
	return nil, nil
}

func lowerIface(src string, node xparse.SyntaxNode[Term, Rule]) (*Decl, error) {
	qualifier, err := lowerQualifiers(src, node)
	if err != nil {
		return nil, err
	}
	decl := qualifier.decl()
	decl.Block = identText(node)
	if decl.Block == "" {
		return nil, syntaxf(node, "block")
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

func lowerFunc(src string, node xparse.SyntaxNode[Term, Rule]) (*Func, error) {
	qualifier, err := lowerQualifiers(src, node)
	if err != nil {
		return nil, err
	}
	if qualifier.storage != 0 || qualifier.invariant || qualifier.precise || qualifier.interp != 0 || qualifier.aux != 0 || len(qualifier.layouts) > 0 || len(qualifier.memory) > 0 {
		return nil, syntaxf(node, "function qualifier")
	}
	fn := &Func{Rank: qualifier.rank}
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
		return nil, syntaxf(node, "function")
	}
	if len(fn.Return.Sizes) == 0 {
		fn.Return.Sizes = nil
	}
	return fn, nil
}

func lowerParams(src string, node xparse.SyntaxNode[Term, Rule]) ([]*Param, error) {
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

func lowerParam(src string, node xparse.SyntaxNode[Term, Rule]) (*Param, error) {
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
			if err := applyParameterQualifier(src, param, child); err != nil {
				return nil, err
			}
		case RuleParameterName:
			name, sizes, err := lowerParameterName(src, child)
			if err != nil {
				return nil, err
			}
			param.Name = name
			param.Type.Sizes = sizes
		}
	}
	if param.Type.Name == "" {
		return nil, syntaxf(node, "parameter")
	}
	return param, nil
}

func applyParameterQualifier(src string, param *Param, node xparse.SyntaxNode[Term, Rule]) error {
	term, text := keyword(node)
	switch term {
	case TermConst:
		param.Const = true
	case TermIn, TermOut, TermInOut:
		storage, err := ParseStorage(text)
		if err != nil {
			return syntaxf(node, "parameter %s", text)
		}
		param.Storage = storage
	case TermLowp, TermMediump, TermHighp:
		rank, err := ParseRank(text)
		if err != nil {
			return syntaxf(node, "parameter %s", text)
		}
		param.Rank = rank
	case TermCoherent, TermVolatile, TermRestrict, TermReadonly, TermWriteonly:
		mem, err := ParseMemory(text)
		if err != nil {
			return syntaxf(node, "parameter %s", text)
		}
		param.Memory = addMemory(param.Memory, mem)
	default:
		return syntaxf(node, "parameter")
	}
	return nil
}

func lowerParameterName(src string, node xparse.SyntaxNode[Term, Rule]) (string, []Size, error) {
	name := identText(node)
	sizes, err := sizesOf(src, node)
	if err != nil {
		return "", nil, err
	}
	return name, sizes, nil
}

func lowerNamed(src string, node xparse.SyntaxNode[Term, Rule]) (*Decl, error) {
	qualifier, err := lowerQualifiers(src, node)
	if err != nil {
		return nil, err
	}
	decl := qualifier.decl()
	decl.Name = identText(node)
	if decl.Name == "" {
		return nil, syntaxf(node, "name")
	}
	return decl, nil
}

func lowerBare(src string, node xparse.SyntaxNode[Term, Rule]) (*Decl, error) {
	qualifier, err := lowerQualifiers(src, node)
	if err != nil {
		return nil, err
	}
	return qualifier.decl(), nil
}

func lowerDecl(src string, node xparse.SyntaxNode[Term, Rule]) ([]*Decl, error) {
	qualifier, err := lowerQualifiers(src, node)
	if err != nil {
		return nil, err
	}
	typeName := identText(node)
	if typeName == "" {
		return nil, syntaxf(node, "declaration")
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleDeclarators {
			return lowerDeclarators(src, child, qualifier, Type{Name: typeName})
		}
	}
	return nil, syntaxf(node, "declaration")
}

func lowerDeclarators(src string, node xparse.SyntaxNode[Term, Rule], qualifier qualifierState, valueType Type) ([]*Decl, error) {
	var decls []*Decl
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleDeclarator {
			continue
		}
		decl, err := lowerDeclarator(src, child, qualifier, valueType)
		if err != nil {
			return nil, err
		}
		decls = append(decls, decl)
	}
	if len(decls) == 0 {
		return nil, syntaxf(node, "declarator")
	}
	return decls, nil
}

func lowerDeclarator(src string, node xparse.SyntaxNode[Term, Rule], qualifier qualifierState, valueType Type) (*Decl, error) {
	decl := qualifier.decl()
	decl.Name = identText(node)
	if decl.Name == "" {
		return nil, syntaxf(node, "declarator")
	}
	sizes, err := sizesOf(src, node)
	if err != nil {
		return nil, err
	}
	decl.Type = withSizes(valueType, sizes)
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

func lowerInit(src string, node xparse.SyntaxNode[Term, Rule]) (Expr, error) {
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
	return nil, syntaxf(node, "initializer")
}

func lowerQualifiers(src string, node xparse.SyntaxNode[Term, Rule]) (qualifierState, error) {
	var qualifier qualifierState
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); !ok || rule != RuleQualifier {
			continue
		}
		if err := applyQual(src, &qualifier, child); err != nil {
			return qualifierState{}, err
		}
	}
	return qualifier, nil
}

func applyQual(src string, qualifier *qualifierState, node xparse.SyntaxNode[Term, Rule]) error {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleLayoutQualifier {
			items, err := lowerLayouts(src, child)
			if err != nil {
				return err
			}
			qualifier.layouts = append(qualifier.layouts, items...)
			continue
		}
		term, text := keyword(child)
		switch term {
		case TermInvariant:
			qualifier.invariant = true
		case TermPrecise:
			qualifier.precise = true
		case TermSmooth, TermFlat, TermNoPerspective:
			interp, err := ParseInterp(text)
			if err != nil || (qualifier.interp != 0 && qualifier.interp != interp) {
				return syntaxf(child, "interpolation %s", text)
			}
			qualifier.interp = interp
		case TermCentroid, TermSample, TermPatch:
			aux, err := ParseAux(text)
			if err != nil || (qualifier.aux != 0 && qualifier.aux != aux) {
				return syntaxf(child, "auxiliary %s", text)
			}
			qualifier.aux = aux
		case TermCoherent, TermVolatile, TermRestrict, TermReadonly, TermWriteonly:
			mem, err := ParseMemory(text)
			if err != nil {
				return syntaxf(child, "memory %s", text)
			}
			qualifier.memory = addMemory(qualifier.memory, mem)
		case TermLowp, TermMediump, TermHighp:
			rank, err := ParseRank(text)
			if err != nil || (qualifier.rank != 0 && qualifier.rank != rank) {
				return syntaxf(child, "precision %s", text)
			}
			qualifier.rank = rank
		case TermConst, TermIn, TermOut, TermInOut, TermUniform, TermBuffer, TermShared:
			storage, err := ParseStorage(text)
			if err != nil || (qualifier.storage != 0 && qualifier.storage != storage) {
				return syntaxf(child, "storage %s", text)
			}
			qualifier.storage = storage
		default:
			if term != "" {
				return syntaxf(child, "qualifier %s", text)
			}
		}
	}
	return nil
}

func lowerLayouts(src string, node xparse.SyntaxNode[Term, Rule]) ([]Layout, error) {
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
					return nil, syntaxf(part, "layout")
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

type qualifierState struct {
	layouts   []Layout
	invariant bool
	precise   bool
	interp    Interp
	aux       Aux
	memory    []Memory
	rank      Rank
	storage   Storage
}

func (qualifier qualifierState) decl() *Decl {
	decl := &Decl{
		Invariant: qualifier.invariant,
		Precise:   qualifier.precise,
		Interp:    qualifier.interp,
		Aux:       qualifier.aux,
		Rank:      qualifier.rank,
		Storage:   qualifier.storage,
	}
	if len(qualifier.layouts) > 0 {
		decl.Layouts = append([]Layout{}, qualifier.layouts...)
	}
	if len(qualifier.memory) > 0 {
		decl.Memory = append([]Memory{}, qualifier.memory...)
	}
	return decl
}

func lowerStmt(src string, node xparse.SyntaxNode[Term, Rule]) (Stmt, error) {
	node = unwrap(node, RuleStatement, RuleSwitchItem)
	rule, ok := node.Rule()
	if !ok {
		return nil, syntaxf(node, "statement")
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
		return nil, syntaxf(node, "statement")
	}
}

func lowerBlockStmts(src string, node xparse.SyntaxNode[Term, Rule]) ([]Stmt, error) {
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

func lowerControlled(src string, node xparse.SyntaxNode[Term, Rule]) ([]Stmt, error) {
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

func lowerIf(src string, node xparse.SyntaxNode[Term, Rule]) (*If, error) {
	item := &If{}
	var stmts []xparse.SyntaxNode[Term, Rule]
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
		return nil, syntaxf(node, "if")
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

func lowerFor(src string, node xparse.SyntaxNode[Term, Rule]) (*For, error) {
	inner, bodyNode, err := parenInner(node)
	if err != nil {
		return nil, syntaxf(node, "for")
	}
	item := &For{}
	if len(inner) == 0 {
		return nil, syntaxf(node, "for")
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
			return nil, syntaxf(node, "for")
		}
		rest = rest[2:]
	default:
		return nil, syntaxf(rest[0], "for")
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

func splitForRest(src string, nodes []xparse.SyntaxNode[Term, Rule]) (Expr, Expr, error) {
	var left, right []xparse.SyntaxNode[Term, Rule]
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
			return nil, nil, syntaxf(xparse.SyntaxNode[Term, Rule]{}, "for")
		}
		return nil, nil, syntaxf(nodes[0], "for")
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

func optionalExpr(src string, nodes []xparse.SyntaxNode[Term, Rule]) (Expr, error) {
	if len(nodes) == 0 {
		return nil, nil
	}
	if len(nodes) != 1 {
		return nil, syntaxf(nodes[0], "expression")
	}
	return lowerExpr(src, nodes[0])
}

func lowerWhile(src string, node xparse.SyntaxNode[Term, Rule]) (*While, error) {
	expr, err := firstExpr(src, node)
	if err != nil {
		return nil, err
	}
	bodyNode := lastStmt(node)
	if !bodyNode.Valid() {
		return nil, syntaxf(node, "while")
	}
	body, err := lowerControlled(src, bodyNode)
	if err != nil {
		return nil, err
	}
	return &While{Cond: expr, Body: body}, nil
}

func lowerDo(src string, node xparse.SyntaxNode[Term, Rule]) (*Do, error) {
	bodyNode := xparse.SyntaxNode[Term, Rule]{}
	var exprNode xparse.SyntaxNode[Term, Rule]
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
		return nil, syntaxf(node, "do")
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

func lowerSwitch(src string, node xparse.SyntaxNode[Term, Rule]) (*Switch, error) {
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
				return nil, syntaxf(child, "case")
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

func lowerReturn(src string, node xparse.SyntaxNode[Term, Rule]) (*Return, error) {
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

func lowerExpr(src string, node xparse.SyntaxNode[Term, Rule]) (Expr, error) {
	if !node.Valid() {
		return nil, syntaxf(xparse.SyntaxNode[Term, Rule]{}, "expression")
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
	return nil, syntaxf(node, "expression")
}

func lowerBinary(src string, left, op, right xparse.SyntaxNode[Term, Rule]) (Expr, error) {
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

func lowerArgs(src string, node xparse.SyntaxNode[Term, Rule]) ([]Expr, error) {
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

func lowerList(src string, node xparse.SyntaxNode[Term, Rule]) (*List, error) {
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

func collectElems(src string, node xparse.SyntaxNode[Term, Rule], out *[]Expr) error {
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

func appendElem(src string, node xparse.SyntaxNode[Term, Rule], out *[]Expr) error {
	node = unwrap(node, RuleElement)
	rule, ok := node.Rule()
	if !ok {
		return syntaxf(node, "initializer")
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

func lowerAtom(src string, node xparse.SyntaxNode[Term, Rule]) (Expr, error) {
	term, ok := node.Term()
	if !ok {
		return nil, syntaxf(node, "expression")
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
		lit, end, err := scanNumber(node.Text(), 0, node.Line())
		if err != nil {
			return nil, err
		}
		if end != len(node.Text()) {
			return nil, syntaxf(node, "number")
		}
		return &lit, nil
	default:
		return nil, syntaxf(node, "expression")
	}
}

func firstExpr(src string, node xparse.SyntaxNode[Term, Rule]) (Expr, error) {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && (rule == RuleExpression || rule == RuleAssign) {
			return lowerExpr(src, child)
		}
	}
	return nil, syntaxf(node, "expression")
}

func lastStmt(node xparse.SyntaxNode[Term, Rule]) xparse.SyntaxNode[Term, Rule] {
	var stmt xparse.SyntaxNode[Term, Rule]
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == RuleStatement {
			stmt = child
		}
	}
	return stmt
}

func parenInner(node xparse.SyntaxNode[Term, Rule]) ([]xparse.SyntaxNode[Term, Rule], xparse.SyntaxNode[Term, Rule], error) {
	var inner []xparse.SyntaxNode[Term, Rule]
	var stmt xparse.SyntaxNode[Term, Rule]
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
		return nil, xparse.SyntaxNode[Term, Rule]{}, fmt.Errorf("%w: for", ErrSyntax)
	}
	return inner, stmt, nil
}

func sizesOf(src string, node xparse.SyntaxNode[Term, Rule]) ([]Size, error) {
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

func lowerBracket(src string, node xparse.SyntaxNode[Term, Rule]) (Size, error) {
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

func withSizes(valueType Type, extra []Size) Type {
	if len(extra) == 0 {
		if len(valueType.Sizes) == 0 {
			return valueType
		}
		valueType.Sizes = append([]Size{}, valueType.Sizes...)
		return valueType
	}
	sizes := make([]Size, 0, len(valueType.Sizes)+len(extra))
	sizes = append(sizes, valueType.Sizes...)
	sizes = append(sizes, extra...)
	valueType.Sizes = sizes
	return valueType
}

func identText(node xparse.SyntaxNode[Term, Rule]) string {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermIdentifier {
			return child.Text()
		}
	}
	return ""
}

func textOf(node xparse.SyntaxNode[Term, Rule]) string {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if _, ok := child.Term(); ok {
			return child.Text()
		}
	}
	return ""
}

func keyword(node xparse.SyntaxNode[Term, Rule]) (Term, string) {
	node = unwrap(node, RuleQualifier, RuleParameterQualifier, RuleRank)
	term, ok := node.Term()
	if !ok {
		return "", ""
	}
	return term, node.Text()
}

func unwrap(node xparse.SyntaxNode[Term, Rule], rules ...Rule) xparse.SyntaxNode[Term, Rule] {
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

func isRule(node xparse.SyntaxNode[Term, Rule], rule Rule) bool {
	got, ok := node.Rule()
	return ok && got == rule
}

func isText(node xparse.SyntaxNode[Term, Rule], text string) bool {
	return node.Valid() && node.Text() == text
}

func syntaxf(node xparse.SyntaxNode[Term, Rule], format string, args ...any) error {
	return fmt.Errorf("%w: line %d: %s", ErrSyntax, node.Line(), fmt.Sprintf(format, args...))
}
