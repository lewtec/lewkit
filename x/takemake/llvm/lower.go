package llvm

import (
	"fmt"
	"strconv"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

func bad(what string) error {
	return fmt.Errorf("%w: %s", ErrSyntax, what)
}

func unsup(what string) error {
	return fmt.Errorf("%w: %s", ErrUnsupported, what)
}

type reader struct {
	n xparse.SyntaxNode[Term, Rule]
	i int
}

func (r *reader) ok() bool { return r.n.Valid() && r.i < r.n.ChildCount() }

func (r *reader) peek() xparse.SyntaxNode[Term, Rule] {
	if !r.ok() {
		return xparse.SyntaxNode[Term, Rule]{}
	}
	return r.n.Child(r.i)
}

func (r *reader) pop() xparse.SyntaxNode[Term, Rule] {
	ch := r.peek()
	if r.ok() {
		r.i++
	}
	return ch
}

func (r *reader) name() string { return r.peek().Name() }

func (r *reader) eat(name string) bool {
	if r.name() != name {
		return false
	}
	r.i++
	return true
}

func (r *reader) rule() (Rule, bool) { return r.peek().Rule() }

func lower(root xparse.SyntaxNode[Term, Rule]) (*File, error) {
	if rule, ok := root.Rule(); !ok || rule != RuleFile {
		return nil, bad("module")
	}
	file := New()
	for i := 0; i < root.ChildCount(); i++ {
		ch := root.Child(i)
		rule, ok := ch.Rule()
		if !ok {
			return nil, bad("module")
		}
		var err error
		switch rule {
		case RuleSource:
			file.SourceFilename, err = decoded(ch, 2)
		case RuleLayout:
			err = lowerLayout(file, ch)
		case RuleModuleAssembly:
			var text string
			text, err = decoded(ch, 2)
			if err == nil {
				file.Asms = append(file.Asms, text)
			}
		case RuleTypedef:
			err = lowerTypeDef(file, ch)
		case RuleGlobalVariable:
			err = lowerGlobal(file, ch)
		case RuleFunction:
			err = lowerFunc(file, ch)
		case RuleTopMetadata:
			err = lowerMeta(file, ch)
		case RuleAttributeGroup:
			err = lowerAttr(file, ch)
		case RuleComdatDefinition:
			err = lowerComdat(file, ch)
		default:
			err = bad("module")
		}
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

func decoded(n xparse.SyntaxNode[Term, Rule], index int) (string, error) {
	if index >= n.ChildCount() {
		return "", bad("string")
	}
	text, ok := decodeString(n.Child(index).Text())
	if !ok {
		return "", bad("string")
	}
	return text, nil
}

func lowerLayout(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	if n.ChildCount() < 4 {
		return bad("target")
	}
	text, ok := decodeString(n.Child(3).Text())
	if !ok {
		return bad("string")
	}
	switch n.Child(1).Name() {
	case "datalayout":
		file.DataLayout = text
	case "triple":
		file.TargetTriple = text
	default:
		return unsup("target")
	}
	return nil
}

func lowerTypeDef(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	name, err := sigilName(r.pop().Text())
	if err != nil {
		return err
	}
	if !r.eat("=") || !r.eat("type") {
		return bad("type")
	}
	def := &TypeDef{Name: name}
	if r.eat("opaque") {
		def.IsOpaque = true
	} else {
		body, err := lowerTypeNode(r.pop())
		if err != nil {
			return err
		}
		def.Body = body
	}
	if r.ok() {
		return bad("type")
	}
	file.Types = append(file.Types, def)
	return nil
}

func lowerComdat(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	name, err := comdatName(r.pop().Text())
	if err != nil {
		return err
	}
	if !r.eat("=") || !r.eat("comdat") {
		return bad("comdat")
	}
	text := r.pop().Text()
	kind, err := ParseComdatKind(text)
	if err != nil {
		return unsup(text)
	}
	file.Comdats = append(file.Comdats, &Comdat{Name: name, Kind: kind})
	return nil
}

func lowerAttr(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	if !r.eat("attributes") || !r.eat("#") {
		return bad("attribute group")
	}
	id := r.pop().Text()
	if !r.eat("=") {
		return bad("attribute group")
	}
	body := r.pop()
	if body.Name() != TermBraced.String() {
		return bad("attribute group")
	}
	file.AttrGroups = append(file.AttrGroups, &AttrGroup{ID: id, Body: body.Text()})
	return nil
}

func lowerMeta(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	if !r.eat(TermMetadataExclamation.String()) {
		return bad("metadata")
	}
	name := r.pop().Text()
	if !r.eat("=") {
		return bad("metadata")
	}
	md := r.pop()
	if rule, ok := md.Rule(); !ok || rule != RuleMetadataNode {
		return bad("metadata")
	}
	file.Metadata = append(file.Metadata, &MetaDef{Name: name, RHS: md.Text()})
	return nil
}

func lowerGlobal(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	name, err := sigilName(r.pop().Text())
	if err != nil {
		return err
	}
	if !r.eat("=") {
		return bad("global")
	}
	g := &Global{Name: name}
	kind, err := takeGlobalPrefix(g, r)
	if err != nil {
		return err
	}
	switch kind {
	case "constant":
		g.IsConstant = true
	case "alias":
		g.Alias = true
	case "ifunc":
		g.IFunc = true
	}
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return err
	}
	g.ValueType = valueType
	if g.Alias || g.IFunc {
		if !r.eat(",") {
			return bad("alias")
		}
		op, err := lowerOperand(r.pop())
		if err != nil {
			return err
		}
		g.Aliasee = &op
	} else if rule, ok := r.rule(); ok && rule == RuleTypedValue {
		init, err := lowerTypedValue(r.pop(), valueType)
		if err != nil {
			return err
		}
		g.Initializer = init
	}
	if err := takeGlobalTrailer(g, r); err != nil {
		return err
	}
	file.Globals = append(file.Globals, g)
	return nil
}

func takeGlobalPrefix(g *Global, r *reader) (string, error) {
	for r.ok() {
		switch r.name() {
		case "global", "constant", "alias", "ifunc":
			return r.pop().Name(), nil
		}
		if rule, ok := r.rule(); ok && rule == RuleWord {
			g.Prefix = append(g.Prefix, r.pop().Text())
			continue
		}
		if link, err := ParseLinkage(r.name()); err == nil {
			if g.Linkage != "" {
				return "", bad("linkage")
			}
			g.Linkage = link
			r.pop()
			continue
		}
		switch r.name() {
		case "dso_local":
			g.DSOLocal = true
			r.pop()
		case "unnamed_addr":
			g.Unnamed = UnnamedAddr
			r.pop()
		case "local_unnamed_addr":
			g.Unnamed = UnnamedLocal
			r.pop()
		default:
			return "", bad("global")
		}
	}
	return "", bad("global")
}

func takeGlobalTrailer(g *Global, r *reader) error {
	for r.eat(",") {
		rule, ok := r.rule()
		if !ok {
			return bad("global")
		}
		switch rule {
		case RulePiece:
			piece := r.pop()
			if n, ok, err := alignOnly(piece); ok || err != nil {
				if err != nil {
					return err
				}
				g.Alignment = n
				continue
			}
			g.Suffix = append(g.Suffix, piece.Text())
		case RuleAttach:
			g.Suffix = append(g.Suffix, r.pop().Text())
		default:
			return bad("global")
		}
	}
	if r.ok() {
		return bad("global")
	}
	return nil
}

func alignOnly(n xparse.SyntaxNode[Term, Rule]) (int, bool, error) {
	if n.ChildCount() != 2 || n.Child(0).Name() != "align" || n.Child(1).Name() != TermNumber.String() {
		return 0, false, nil
	}
	v, err := integer(n.Child(1).Text())
	if err != nil {
		return 0, false, err
	}
	return int(v), true, nil
}

func lowerFunc(file *File, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	fn := &Func{Defined: r.eat("define")}
	if !fn.Defined && !r.eat("declare") {
		return bad("function")
	}
	header := r.pop()
	if rule, ok := header.Rule(); !ok || rule != RuleHeader {
		return bad("function")
	}
	if err := lowerHeader(fn, header); err != nil {
		return err
	}
	if fn.Defined {
		body := r.pop()
		if rule, ok := body.Rule(); !ok || rule != RuleBody {
			return bad("function")
		}
		if err := lowerBody(fn, body); err != nil {
			return err
		}
	}
	if r.ok() {
		return bad("function")
	}
	file.Funcs = append(file.Funcs, fn)
	return nil
}

func lowerHeader(fn *Func, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	for {
		ch := r.peek()
		if rule, ok := ch.Rule(); ok && rule == RuleType {
			break
		}
		if !r.ok() {
			return bad("function")
		}
		if rule, ok := ch.Rule(); ok {
			switch rule {
			case RuleWord:
				if resultAttr(ch) {
					for {
						next, ok := r.rule()
						if !ok || next != RuleWord {
							break
						}
						fn.ResultAttrs = append(fn.ResultAttrs, r.pop().Text())
					}
					continue
				}
				fn.Prefix = append(fn.Prefix, r.pop().Text())
			case RuleCallingConventionNumber:
				fn.Conv = r.pop().Text()
			default:
				return bad("function")
			}
			continue
		}
		if link, err := ParseLinkage(ch.Name()); err == nil {
			if fn.Linkage != "" {
				return bad("linkage")
			}
			fn.Linkage = link
			r.pop()
			continue
		}
		switch ch.Name() {
		case "dso_local":
			fn.DSOLocal = true
			r.pop()
		case "unnamed_addr":
			fn.Unnamed = UnnamedAddr
			r.pop()
		case "local_unnamed_addr":
			fn.Unnamed = UnnamedLocal
			r.pop()
		default:
			if isConv(ch.Name()) {
				fn.Conv = r.pop().Text()
				continue
			}
			return bad("function")
		}
	}
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return err
	}
	fn.Result = valueType
	if r.name() != TermGlobalReference.String() {
		return bad("function name")
	}
	fn.Name, err = sigilName(r.pop().Text())
	if err != nil {
		return err
	}
	params, variadic, err := lowerParams(r.pop())
	if err != nil {
		return err
	}
	fn.Parameters = params
	fn.Variadic = variadic
	for r.ok() {
		ch := r.pop()
		switch ch.Name() {
		case "unnamed_addr":
			fn.Unnamed = UnnamedAddr
			continue
		case "local_unnamed_addr":
			fn.Unnamed = UnnamedLocal
			continue
		}
		rule, ok := ch.Rule()
		if !ok {
			return bad("function suffix")
		}
		switch rule {
		case RuleAttach:
			att, err := lowerAttach(ch)
			if err != nil {
				return err
			}
			fn.Attach = append(fn.Attach, att)
		case RulePiece:
			fn.Suffix = append(fn.Suffix, ch.Text())
		default:
			return bad("function suffix")
		}
	}
	return nil
}

func resultAttr(n xparse.SyntaxNode[Term, Rule]) bool {
	if n.ChildCount() == 0 {
		return false
	}
	head := n.Child(0)
	if isAttr(head.Name()) || isAttr(head.Text()) {
		return true
	}
	word := head.Text()
	if i := strings.IndexByte(word, '('); i > 0 {
		return isAttr(word[:i])
	}
	return false
}

func lowerParams(n xparse.SyntaxNode[Term, Rule]) ([]*Param, bool, error) {
	r := &reader{n: n}
	if !r.eat("(") {
		return nil, false, bad("parameter")
	}
	var params []*Param
	variadic := false
	if !r.eat(")") {
		for {
			if r.eat(TermEllipsis.String()) {
				variadic = true
				break
			}
			if rule, ok := r.rule(); !ok || rule != RuleParameter {
				return nil, false, bad("parameter")
			}
			param, err := lowerParam(r.pop())
			if err != nil {
				return nil, false, err
			}
			params = append(params, param)
			if !r.eat(",") {
				break
			}
		}
		if !r.eat(")") {
			return nil, false, bad("parameter")
		}
	}
	return params, variadic, nil
}

func lowerParam(n xparse.SyntaxNode[Term, Rule]) (*Param, error) {
	r := &reader{n: n}
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	param := &Param{Type: valueType}
	for {
		rule, ok := r.rule()
		if !ok || rule != RuleWord {
			break
		}
		param.Attrs = append(param.Attrs, r.pop().Text())
	}
	if r.name() == TermLocalReference.String() {
		param.Name, err = sigilName(r.pop().Text())
		if err != nil {
			return nil, err
		}
	}
	if r.ok() {
		return nil, bad("parameter")
	}
	return param, nil
}

func lowerBody(fn *Func, n xparse.SyntaxNode[Term, Rule]) error {
	var block *Block
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		switch ch.Name() {
		case "{", "}":
			continue
		case TermLabelName.String():
			block = &Block{Name: ch.Text()}
			fn.Blocks = append(fn.Blocks, block)
			continue
		}
		if rule, ok := ch.Rule(); !ok || rule != RuleInstruction {
			return bad("instruction")
		}
		if block == nil {
			block = &Block{}
			fn.Blocks = append(fn.Blocks, block)
		}
		in, err := lowerInst(ch)
		if err != nil {
			return err
		}
		block.Instructions = append(block.Instructions, in)
	}
	return nil
}

func lowerInst(n xparse.SyntaxNode[Term, Rule]) (*Inst, error) {
	in := &Inst{}
	r := &reader{n: n}
	if r.name() == TermLocalDefinition.String() {
		var err error
		in.Name, err = sigilName(r.pop().Text())
		if err != nil {
			return nil, err
		}
		if !r.eat("=") {
			return nil, bad("instruction")
		}
	}
	op := r.pop()
	switch op.Name() {
	case "tail", "musttail", "notail":
		in.Tail = Tail(op.Name())
		if !r.eat("call") {
			return nil, bad("call")
		}
		in.Kind = InstCall
		if err := fillCall(in, r); err != nil {
			return nil, err
		}
		return finish(in, r, false)
	case "call":
		in.Kind = InstCall
		if err := fillCall(in, r); err != nil {
			return nil, err
		}
		return finish(in, r, false)
	case "invoke":
		in.Kind = InstInvoke
		if err := fillCall(in, r); err != nil {
			return nil, err
		}
		if err := readInvoke(in, r); err != nil {
			return nil, err
		}
		return finish(in, r, false)
	case "callbr":
		in.Kind = InstCallBr
		if err := fillCall(in, r); err != nil {
			return nil, err
		}
		if err := readCallBr(in, r); err != nil {
			return nil, err
		}
		return finish(in, r, false)
	case "alloca":
		return lowerAlloca(in, r)
	case "load":
		return lowerLoad(in, r)
	case "store":
		return lowerStore(in, r)
	case "getelementptr":
		return lowerGEPInst(in, r)
	case "icmp":
		return lowerICmp(in, r)
	case "fcmp":
		return lowerFCmp(in, r)
	case "phi":
		return lowerPhi(in, r)
	case "extractvalue":
		return lowerExtract(in, r)
	case "insertvalue":
		return lowerInsert(in, r)
	case "br":
		return lowerBr(in, r)
	case "ret":
		return lowerRet(in, r)
	case "unreachable":
		in.Kind = InstUnreachable
		return finish(in, r, false)
	case "fneg":
		return lowerFNeg(in, r)
	case "freeze":
		return lowerFreeze(in, r)
	case "select":
		return lowerSelect(in, r)
	case "extractelement":
		return lowerExtractElt(in, r)
	case "insertelement":
		return lowerInsertElt(in, r)
	case "shufflevector":
		return lowerShuffle(in, r)
	case "atomicrmw":
		return lowerAtomicRMW(in, r)
	case "cmpxchg":
		return lowerCmpXchg(in, r)
	case "fence":
		return lowerFence(in, r)
	case "switch":
		return lowerSwitch(in, r)
	case "resume":
		return lowerResume(in, r)
	case "landingpad":
		return lowerLanding(in, r)
	case "indirectbr":
		return lowerIndirect(in, r)
	case "va_arg":
		return lowerVAArg(in, r)
	default:
		if cast, err := ParseCast(op.Name()); err == nil {
			return lowerCast(in, cast, r)
		}
		if bin, err := ParseOp(op.Name()); err == nil {
			return lowerBin(in, bin, r)
		}
		return nil, unsup(op.Name())
	}
}

func finish(in *Inst, r *reader, align bool) (*Inst, error) {
	for r.eat(",") {
		if align && r.eat("align") {
			n, err := integer(r.pop().Text())
			if err != nil {
				return nil, err
			}
			in.Alignment = int(n)
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleAttach {
			att, err := lowerAttach(r.pop())
			if err != nil {
				return nil, err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		return nil, bad("instruction")
	}
	if r.ok() {
		return nil, bad("instruction")
	}
	return in, nil
}

func lowerAlloca(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstAlloca
	if r.eat("inalloca") {
		in.Flags = append(in.Flags, "inalloca")
	}
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	for r.eat(",") {
		if r.eat("align") {
			n, err := integer(r.pop().Text())
			if err != nil {
				return nil, err
			}
			in.Alignment = int(n)
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleAttach {
			att, err := lowerAttach(r.pop())
			if err != nil {
				return nil, err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleOperand {
			if in.Count != nil {
				return nil, bad("alloca")
			}
			op, err := lowerOperand(r.pop())
			if err != nil {
				return nil, err
			}
			in.Count = &op
			continue
		}
		return nil, bad("alloca")
	}
	if r.ok() {
		return nil, bad("alloca")
	}
	return in, nil
}

func lowerLoad(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstLoad
	if r.eat("atomic") {
		in.Atomic = true
	}
	if r.eat("volatile") {
		in.IsVolatile = true
	}
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	if !r.eat(",") {
		return nil, bad("load")
	}
	op, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Pointer = op.Type
	in.Left = op.Value
	if err := takeAtomic(in, r); err != nil {
		return nil, err
	}
	return finish(in, r, true)
}

func lowerStore(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstStore
	if r.eat("atomic") {
		in.Atomic = true
	}
	if r.eat("volatile") {
		in.IsVolatile = true
	}
	val, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = val.Type
	in.Left = val.Value
	if !r.eat(",") {
		return nil, bad("store")
	}
	ptr, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Pointer = ptr.Type
	in.Right = ptr.Value
	if err := takeAtomic(in, r); err != nil {
		return nil, err
	}
	return finish(in, r, true)
}

func takeAtomic(in *Inst, r *reader) error {
	var scope, ord string
	if rule, ok := r.rule(); ok && rule == RuleSync {
		scope = r.pop().Text()
	}
	if isOrdering(r.name()) {
		ord = r.pop().Text()
	}
	if !in.Atomic {
		if scope != "" || ord != "" {
			return bad("ordering")
		}
		return nil
	}
	if ord == "" {
		return bad("ordering")
	}
	in.SyncScope = scope
	in.Ordering = ord
	return nil
}

func isOrdering(name string) bool {
	switch name {
	case "unordered", "monotonic", "acquire", "release", "acq_rel", "seq_cst":
		return true
	default:
		return false
	}
}

func lowerGEPInst(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstGEP
	takeInstGEPFlags(in, r)
	src, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Source = src
	if !r.eat(",") {
		return nil, bad("getelementptr")
	}
	base, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Pointer = base.Type
	in.Left = base.Value
	for r.eat(",") {
		if r.eat("align") {
			n, err := integer(r.pop().Text())
			if err != nil {
				return nil, err
			}
			in.Alignment = int(n)
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleAttach {
			att, err := lowerAttach(r.pop())
			if err != nil {
				return nil, err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleOperand {
			idx, err := lowerOperand(r.pop())
			if err != nil {
				return nil, err
			}
			in.Indices = append(in.Indices, idx)
			continue
		}
		return nil, bad("getelementptr")
	}
	if r.ok() {
		return nil, bad("getelementptr")
	}
	return in, nil
}

func takeInstGEPFlags(in *Inst, r *reader) {
	for {
		switch r.name() {
		case "inbounds":
			in.InBounds = true
			r.pop()
		case "nuw":
			in.NoUnsignedWrap = true
			r.pop()
		case "nsw":
			in.NoSignedWrap = true
			r.pop()
		case "nusw", "inrange":
			in.Flags = append(in.Flags, r.pop().Text())
		default:
			return
		}
	}
}

func lowerBin(in *Inst, op Op, r *reader) (*Inst, error) {
	in.Kind = InstBin
	in.Op = op
	for {
		switch r.name() {
		case "nuw":
			in.NoUnsignedWrap = true
			r.pop()
		case "nsw":
			in.NoSignedWrap = true
			r.pop()
		case "exact":
			in.ExactDiv = true
			r.pop()
		default:
			if !isFastMath(r.name()) && r.name() != "disjoint" {
				goto values
			}
			in.Flags = append(in.Flags, r.pop().Text())
		}
	}
values:
	valueType, left, right, err := cmpValues(r)
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	in.Left = left
	in.Right = right
	return finish(in, r, false)
}

func lowerICmp(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstICmp
	for r.name() == TermIdentifier.String() {
		in.Flags = append(in.Flags, r.pop().Text())
	}
	pred, err := ParseIPred(r.pop().Text())
	if err != nil {
		return nil, unsup("predicate")
	}
	in.IntegerPred = pred
	valueType, left, right, err := cmpValues(r)
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	in.Left = left
	in.Right = right
	return finish(in, r, false)
}

func lowerFCmp(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstFCmp
	for isFastMath(r.name()) {
		in.Flags = append(in.Flags, r.pop().Text())
	}
	pred, err := ParseFPred(r.pop().Text())
	if err != nil {
		return nil, unsup("predicate")
	}
	in.FloatPred = pred
	valueType, left, right, err := cmpValues(r)
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	in.Left = left
	in.Right = right
	return finish(in, r, false)
}

func cmpValues(r *reader) (*Type, *Value, *Value, error) {
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, nil, nil, err
	}
	left, err := lowerValue(r.pop(), valueType)
	if err != nil {
		return nil, nil, nil, err
	}
	if !r.eat(",") {
		return nil, nil, nil, bad("instruction")
	}
	right, err := lowerValue(r.pop(), valueType)
	if err != nil {
		return nil, nil, nil, err
	}
	return valueType, left, right, nil
}

func lowerCast(in *Inst, kind Cast, r *reader) (*Inst, error) {
	in.Kind = InstCast
	in.Cast = kind
	for {
		switch r.name() {
		case "nuw":
			in.NoUnsignedWrap = true
			r.pop()
		case "nsw":
			in.NoSignedWrap = true
			r.pop()
		default:
			if !isFastMath(r.name()) && r.name() != "nneg" {
				goto body
			}
			in.Flags = append(in.Flags, r.pop().Text())
		}
	}
body:
	from, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = from
	val, err := lowerTypedValue(r.pop(), from)
	if err != nil {
		return nil, err
	}
	in.Left = val
	if !r.eat("to") {
		return nil, bad("cast")
	}
	dest, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Dest = dest
	return finish(in, r, false)
}

func lowerPhi(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstPhi
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	arm, err := readPhi(r, valueType)
	if err != nil {
		return nil, err
	}
	in.Incoming = append(in.Incoming, arm)
	for r.eat(",") {
		if r.name() == "[" {
			arm, err = readPhi(r, valueType)
			if err != nil {
				return nil, err
			}
			in.Incoming = append(in.Incoming, arm)
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleAttach {
			att, err := lowerAttach(r.pop())
			if err != nil {
				return nil, err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		return nil, bad("phi")
	}
	if r.ok() {
		return nil, bad("phi")
	}
	return in, nil
}

func readPhi(r *reader, valueType *Type) (Incoming, error) {
	if !r.eat("[") {
		return Incoming{}, bad("phi")
	}
	val, err := lowerValue(r.pop(), valueType)
	if err != nil {
		return Incoming{}, err
	}
	if !r.eat(",") || r.name() != TermLocalReference.String() {
		return Incoming{}, bad("phi")
	}
	block, err := sigilName(r.pop().Text())
	if err != nil {
		return Incoming{}, err
	}
	if !r.eat("]") {
		return Incoming{}, bad("phi")
	}
	return Incoming{Value: val, Block: block}, nil
}

func lowerExtract(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstExtract
	op, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = op.Type
	in.Left = op.Value
	path, err := readPath(in, r)
	if err != nil {
		return nil, err
	}
	in.Path = path
	if r.ok() {
		return nil, bad("extractvalue")
	}
	return in, nil
}

func lowerInsert(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstInsert
	agg, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = agg.Type
	in.Left = agg.Value
	if !r.eat(",") {
		return nil, bad("insertvalue")
	}
	elem, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Inserted = &elem
	path, err := readPath(in, r)
	if err != nil {
		return nil, err
	}
	in.Path = path
	if r.ok() {
		return nil, bad("insertvalue")
	}
	return in, nil
}

func readPath(in *Inst, r *reader) ([]int, error) {
	var path []int
	for r.eat(",") {
		if r.name() == TermNumber.String() {
			n, err := integer(r.pop().Text())
			if err != nil {
				return nil, err
			}
			path = append(path, int(n))
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleAttach {
			att, err := lowerAttach(r.pop())
			if err != nil {
				return nil, err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		return nil, bad("index")
	}
	if len(path) == 0 {
		return nil, bad("index")
	}
	return path, nil
}

func lowerBr(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstBr
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	val, err := lowerTypedValue(r.pop(), IntType(1))
	if err != nil {
		return nil, err
	}
	branched := false
	for r.eat(",") {
		if r.name() == "label" {
			if branched {
				return nil, bad("br")
			}
			r.pop()
			if r.name() != TermLocalReference.String() {
				return nil, bad("label")
			}
			in.TrueLabel, err = sigilName(r.pop().Text())
			if err != nil {
				return nil, err
			}
			if !r.eat(",") || !r.eat("label") || r.name() != TermLocalReference.String() {
				return nil, bad("label")
			}
			in.FalseLabel, err = sigilName(r.pop().Text())
			if err != nil {
				return nil, err
			}
			branched = true
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleAttach {
			att, err := lowerAttach(r.pop())
			if err != nil {
				return nil, err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		return nil, bad("br")
	}
	if r.ok() {
		return nil, bad("br")
	}
	if !branched {
		if valueType.Kind != KindLabel || val.Kind != ValLocal || len(val.Attrs) > 0 {
			return nil, bad("label")
		}
		in.Target = val.Name
		return in, nil
	}
	in.Condition = val
	return in, nil
}

func lowerRet(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstRet
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	if rule, ok := r.rule(); ok && rule == RuleTypedValue {
		if valueType.Kind == KindVoid {
			return nil, bad("ret")
		}
		val, err := lowerTypedValue(r.pop(), valueType)
		if err != nil {
			return nil, err
		}
		in.Left = val
	} else if valueType.Kind != KindVoid {
		return nil, bad("ret")
	}
	return finish(in, r, false)
}

func lowerFNeg(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstFNeg
	for isFastMath(r.name()) {
		in.Flags = append(in.Flags, r.pop().Text())
	}
	return lowerUnary(in, r)
}

func lowerFreeze(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstFreeze
	return lowerUnary(in, r)
}

func lowerUnary(in *Inst, r *reader) (*Inst, error) {
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	val, err := lowerTypedValue(r.pop(), valueType)
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	in.Left = val
	return finish(in, r, false)
}

func lowerSelect(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstSelect
	for isFastMath(r.name()) {
		in.Flags = append(in.Flags, r.pop().Text())
	}
	cond, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("select")
	}
	left, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("select")
	}
	if _, err := lowerTypeNode(r.pop()); err != nil {
		return nil, err
	}
	right, err := lowerTypedValue(r.pop(), left.Type)
	if err != nil {
		return nil, err
	}
	in.CondType = cond.Type
	in.Condition = cond.Value
	in.Type = left.Type
	in.Left = left.Value
	in.Right = right
	return finish(in, r, false)
}

func lowerExtractElt(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstExtractElt
	vec, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("extractelement")
	}
	idx, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = vec.Type
	in.Left = vec.Value
	in.Index = &idx
	return finish(in, r, false)
}

func lowerInsertElt(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstInsertElt
	vec, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("insertelement")
	}
	elem, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("insertelement")
	}
	idx, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = vec.Type
	in.Left = vec.Value
	in.Inserted = &elem
	in.Index = &idx
	return finish(in, r, false)
}

func lowerShuffle(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstShuffle
	left, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("shufflevector")
	}
	other, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	if !r.eat(",") {
		return nil, bad("shufflevector")
	}
	mask, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = left.Type
	in.Left = left.Value
	in.Other = &other
	in.Index = &mask
	return finish(in, r, false)
}

func lowerAtomicRMW(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstAtomicRMW
	if r.eat("volatile") {
		in.IsVolatile = true
	}
	if rule, ok := r.rule(); ok && rule == RuleSync {
		in.SyncScope = r.pop().Text()
	}
	if !isAtomicOp(r.name()) {
		return nil, unsup(r.name())
	}
	in.AtomicOp = r.pop().Text()
	ptr, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Pointer = ptr.Type
	in.Left = ptr.Value
	if !r.eat(",") {
		return nil, bad("atomicrmw")
	}
	val, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = val.Type
	in.Right = val.Value
	if !isOrdering(r.name()) {
		return nil, bad("ordering")
	}
	in.Ordering = r.pop().Text()
	return finish(in, r, true)
}

func lowerCmpXchg(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstCmpXchg
	for r.name() == "weak" || r.name() == "volatile" {
		if r.eat("weak") {
			in.Weak = true
			continue
		}
		r.pop()
		in.IsVolatile = true
	}
	if rule, ok := r.rule(); ok && rule == RuleSync {
		in.SyncScope = r.pop().Text()
	}
	ptr, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Pointer = ptr.Type
	in.Left = ptr.Value
	if !r.eat(",") {
		return nil, bad("cmpxchg")
	}
	exp, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = exp.Type
	in.Right = exp.Value
	if !r.eat(",") {
		return nil, bad("cmpxchg")
	}
	neu, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Inserted = &neu
	if !isOrdering(r.name()) {
		return nil, bad("ordering")
	}
	in.Ordering = r.pop().Text()
	if !isOrdering(r.name()) {
		return nil, bad("ordering")
	}
	in.FailureOrd = r.pop().Text()
	return finish(in, r, true)
}

func lowerFence(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstFence
	if rule, ok := r.rule(); ok && rule == RuleSync {
		in.SyncScope = r.pop().Text()
	}
	if !isOrdering(r.name()) {
		return nil, bad("ordering")
	}
	in.Ordering = r.pop().Text()
	return finish(in, r, false)
}

func lowerSwitch(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstSwitch
	val, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = val.Type
	in.Left = val.Value
	if !r.eat(",") || !r.eat("label") || r.name() != TermLocalReference.String() {
		return nil, bad("switch")
	}
	in.Target, err = sigilName(r.pop().Text())
	if err != nil {
		return nil, err
	}
	if !r.eat("[") {
		return nil, bad("switch")
	}
	for r.name() != "]" && r.ok() {
		armType, err := lowerOperand(r.pop())
		if err != nil {
			return nil, err
		}
		if !r.eat(",") || !r.eat("label") || r.name() != TermLocalReference.String() {
			return nil, bad("switch")
		}
		label, err := sigilName(r.pop().Text())
		if err != nil {
			return nil, err
		}
		in.Arms = append(in.Arms, Arm{Type: armType.Type, Value: armType.Value, Label: label})
	}
	if !r.eat("]") {
		return nil, bad("switch")
	}
	return finish(in, r, false)
}

func lowerResume(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstResume
	op, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = op.Type
	in.Left = op.Value
	return finish(in, r, false)
}

func lowerLanding(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstLanding
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = valueType
	for {
		switch r.name() {
		case "cleanup":
			r.pop()
			in.Clauses = append(in.Clauses, Clause{Kind: "cleanup"})
		case "catch", "filter":
			kind := r.pop().Name()
			op, err := lowerOperand(r.pop())
			if err != nil {
				return nil, err
			}
			in.Clauses = append(in.Clauses, Clause{Kind: kind, Type: op.Type, Value: op.Value})
		default:
			return finish(in, r, false)
		}
	}
}

func lowerIndirect(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstIndirectBr
	op, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Type = op.Type
	in.Left = op.Value
	if !r.eat(",") || !r.eat("[") {
		return nil, bad("indirectbr")
	}
	for r.name() == "label" {
		r.pop()
		if r.name() != TermLocalReference.String() {
			return nil, bad("label")
		}
		label, err := sigilName(r.pop().Text())
		if err != nil {
			return nil, err
		}
		in.Labels = append(in.Labels, label)
		r.eat(",")
	}
	if !r.eat("]") {
		return nil, bad("indirectbr")
	}
	return finish(in, r, false)
}

func lowerVAArg(in *Inst, r *reader) (*Inst, error) {
	in.Kind = InstVAArg
	op, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	in.Pointer = op.Type
	in.Left = op.Value
	if !r.eat(",") {
		return nil, bad("va_arg")
	}
	dest, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	in.Dest = dest
	in.Type = dest
	return finish(in, r, false)
}

func fillCall(in *Inst, r *reader) error {
	if rule, ok := r.rule(); !ok || rule != RuleCallBody {
		return bad("call")
	}
	return lowerCallBody(in, r.pop())
}

func lowerCallBody(in *Inst, n xparse.SyntaxNode[Term, Rule]) error {
	r := &reader{n: n}
	for isFastMath(r.name()) {
		in.Flags = append(in.Flags, r.pop().Text())
	}
	if rule, ok := r.rule(); ok && rule == RuleCallingConventionNumber {
		in.Conv = r.pop().Text()
	} else if isConv(r.name()) {
		in.Conv = r.pop().Text()
	}
	for {
		rule, ok := r.rule()
		if !ok || rule != RuleWord {
			break
		}
		in.ResultAttrs = append(in.ResultAttrs, r.pop().Text())
	}
	valueType, err := lowerTypeNode(r.pop())
	if err != nil {
		return err
	}
	if valueType.Kind == KindFunc {
		in.Signature = valueType
		in.Type = valueType.Ret
	} else {
		in.Type = valueType
	}
	callee, err := lowerTypedValue(r.pop(), nil)
	if err != nil {
		return err
	}
	in.Callee = callee
	if !r.eat("(") {
		return bad("call")
	}
	if !r.eat(")") {
		for {
			op, err := lowerOperand(r.pop())
			if err != nil {
				return err
			}
			in.Args = append(in.Args, op)
			if !r.eat(",") {
				break
			}
		}
		if !r.eat(")") {
			return bad("call")
		}
	}
	for r.ok() {
		switch r.name() {
		case "#":
			r.pop()
			in.Groups = append(in.Groups, "#"+r.pop().Text())
		case TermIdentifier.String(), TermAttributeGroupReference.String():
			in.Groups = append(in.Groups, r.pop().Text())
		default:
			return bad("call")
		}
	}
	return nil
}

func readInvoke(in *Inst, r *reader) error {
	if !r.eat("to") || !r.eat("label") || r.name() != TermLocalReference.String() {
		return bad("invoke")
	}
	var err error
	in.TrueLabel, err = sigilName(r.pop().Text())
	if err != nil {
		return err
	}
	if !r.eat("unwind") || !r.eat("label") || r.name() != TermLocalReference.String() {
		return bad("invoke")
	}
	in.FalseLabel, err = sigilName(r.pop().Text())
	return err
}

func readCallBr(in *Inst, r *reader) error {
	if !r.eat("to") || !r.eat("label") || r.name() != TermLocalReference.String() {
		return bad("callbr")
	}
	var err error
	in.TrueLabel, err = sigilName(r.pop().Text())
	if err != nil {
		return err
	}
	if !r.eat("[") {
		return bad("callbr")
	}
	for r.name() == "label" {
		r.pop()
		if r.name() != TermLocalReference.String() {
			return bad("label")
		}
		label, err := sigilName(r.pop().Text())
		if err != nil {
			return err
		}
		in.Labels = append(in.Labels, label)
		r.eat(",")
	}
	if !r.eat("]") {
		return bad("callbr")
	}
	return nil
}

func lowerAttach(n xparse.SyntaxNode[Term, Rule]) (Attach, error) {
	if n.ChildCount() < 3 {
		return Attach{}, bad("metadata")
	}
	return Attach{Name: n.Child(1).Text(), Body: n.Child(2).Text()}, nil
}

func lowerTypeNode(n xparse.SyntaxNode[Term, Rule]) (*Type, error) {
	rule, ok := n.Rule()
	if !ok || (rule != RuleType && rule != RuleTypeWithoutStruct) {
		return nil, bad("type")
	}
	r := &reader{n: n}
	valueType, err := lowerCore(r)
	if err != nil {
		return nil, err
	}
	for r.ok() {
		if r.name() == "*" {
			valueType = &Type{Kind: KindPtr, Elem: valueType}
			r.pop()
			continue
		}
		if rule, ok := r.rule(); ok && rule == RuleStar {
			valueType, err = wrapStar(valueType, r.pop())
			if err != nil {
				return nil, err
			}
			continue
		}
		break
	}
	if rule, ok := r.rule(); ok && rule == RuleFunctionTail {
		valueType, err = applyFunction(valueType, r.pop())
		if err != nil {
			return nil, err
		}
	}
	if r.ok() {
		return nil, bad("type")
	}
	return valueType, nil
}

func lowerCore(r *reader) (*Type, error) {
	switch r.name() {
	case "ptr":
		r.pop()
		valueType := Ptr()
		if rule, ok := r.rule(); ok && rule == RuleAddressSpace {
			space, err := addrOf(r.pop())
			if err != nil {
				return nil, err
			}
			valueType.AddrSpace = space
		}
		return valueType, nil
	case TermIntegerType.String():
		bits, ok := intBits(r.pop().Text())
		if !ok {
			return nil, bad("type")
		}
		return IntType(bits), nil
	case TermLocalReference.String():
		name, err := sigilName(r.pop().Text())
		if err != nil {
			return nil, err
		}
		return Named(name), nil
	case "void":
		r.pop()
		return Void(), nil
	case "half":
		r.pop()
		return FP(FloatHalf), nil
	case "float":
		r.pop()
		return FP(FloatFloat), nil
	case "double":
		r.pop()
		return FP(FloatDouble), nil
	case "metadata":
		r.pop()
		return &Type{Kind: KindMeta}, nil
	case "token":
		r.pop()
		return &Type{Kind: KindToken}, nil
	case "label":
		r.pop()
		return &Type{Kind: KindLabel}, nil
	case "bfloat", "x86_fp80", "fp128", "ppc_fp128", "x86_mmx", "x86_amx":
		return &Type{Kind: KindPrim, Name: r.pop().Text()}, nil
	case "{":
		r.pop()
		fields, err := readTypes(r, "}")
		if err != nil {
			return nil, err
		}
		return Struct(fields...), nil
	case "[":
		r.pop()
		n, err := integer(r.pop().Text())
		if err != nil {
			return nil, err
		}
		if r.pop().Text() != "x" {
			return nil, bad("type")
		}
		elem, err := lowerTypeNode(r.pop())
		if err != nil {
			return nil, err
		}
		if !r.eat("]") {
			return nil, bad("type")
		}
		return Array(n, elem), nil
	case "<":
		r.pop()
		switch r.name() {
		case "{":
			r.pop()
			fields, err := readTypes(r, "}")
			if err != nil {
				return nil, err
			}
			if !r.eat(">") {
				return nil, bad("type")
			}
			return PackedStruct(fields...), nil
		case TermIdentifier.String():
			if r.pop().Text() != "vscale" || r.pop().Text() != "x" {
				return nil, bad("type")
			}
			n, err := integer(r.pop().Text())
			if err != nil {
				return nil, err
			}
			if r.pop().Text() != "x" {
				return nil, bad("type")
			}
			elem, err := lowerTypeNode(r.pop())
			if err != nil {
				return nil, err
			}
			if !r.eat(">") {
				return nil, bad("type")
			}
			valueType := Vector(n, elem)
			valueType.Scalable = true
			return valueType, nil
		case TermNumber.String():
			n, err := integer(r.pop().Text())
			if err != nil {
				return nil, err
			}
			if r.pop().Text() != "x" {
				return nil, bad("type")
			}
			elem, err := lowerTypeNode(r.pop())
			if err != nil {
				return nil, err
			}
			if !r.eat(">") {
				return nil, bad("type")
			}
			return Vector(n, elem), nil
		default:
			return nil, bad("type")
		}
	default:
		return nil, bad("type")
	}
}

func readTypes(r *reader, close string) ([]*Type, error) {
	if r.eat(close) {
		return nil, nil
	}
	var fields []*Type
	for {
		valueType, err := lowerTypeNode(r.pop())
		if err != nil {
			return nil, err
		}
		fields = append(fields, valueType)
		if !r.eat(",") {
			break
		}
	}
	if !r.eat(close) {
		return nil, bad("type")
	}
	return fields, nil
}

func wrapStar(elem *Type, n xparse.SyntaxNode[Term, Rule]) (*Type, error) {
	space := 0
	if n.ChildCount() >= 2 {
		var err error
		space, err = addrOf(n.Child(0))
		if err != nil {
			return nil, err
		}
	}
	return &Type{Kind: KindPtr, Elem: elem, AddrSpace: space}, nil
}

func addrOf(n xparse.SyntaxNode[Term, Rule]) (int, error) {
	r := &reader{n: n}
	if !r.eat("addrspace") || !r.eat("(") {
		return 0, bad("addrspace")
	}
	v, err := integer(r.pop().Text())
	if err != nil {
		return 0, err
	}
	if !r.eat(")") {
		return 0, bad("addrspace")
	}
	return int(v), nil
}

func applyFunction(ret *Type, n xparse.SyntaxNode[Term, Rule]) (*Type, error) {
	r := &reader{n: n}
	if !r.eat("(") {
		return nil, bad("type")
	}
	var params []*Type
	variadic := false
	if !r.eat(")") {
		for {
			if r.eat(TermEllipsis.String()) {
				variadic = true
				break
			}
			valueType, err := lowerTypeNode(r.pop())
			if err != nil {
				return nil, err
			}
			params = append(params, valueType)
			if !r.eat(",") {
				break
			}
		}
		if !r.eat(")") {
			return nil, bad("type")
		}
	}
	valueType := &Type{Kind: KindFunc, Ret: ret, Fields: params, Variadic: variadic}
	for {
		rule, ok := r.rule()
		if !ok || rule != RuleStar {
			break
		}
		var err error
		valueType, err = wrapStar(valueType, r.pop())
		if err != nil {
			return nil, err
		}
	}
	if r.ok() {
		return nil, bad("type")
	}
	return valueType, nil
}

func lowerOperand(n xparse.SyntaxNode[Term, Rule]) (Operand, error) {
	rule, ok := n.Rule()
	if !ok || (rule != RuleOperand && rule != RuleOperandWithoutStruct) {
		return Operand{}, bad("value")
	}
	if n.ChildCount() < 2 {
		return Operand{}, bad("value")
	}
	valueType, err := lowerTypeNode(n.Child(0))
	if err != nil {
		return Operand{}, err
	}
	val, err := lowerTypedValue(n.Child(1), valueType)
	if err != nil {
		return Operand{}, err
	}
	return Operand{Type: valueType, Value: val}, nil
}

func lowerTypedValue(n xparse.SyntaxNode[Term, Rule], ctx *Type) (*Value, error) {
	if rule, ok := n.Rule(); !ok || rule != RuleTypedValue {
		return nil, bad("value")
	}
	var attrs []string
	var val xparse.SyntaxNode[Term, Rule]
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		if rule, ok := ch.Rule(); ok && rule == RuleWord {
			attrs = append(attrs, ch.Text())
			continue
		}
		val = ch
	}
	out, err := lowerValue(val, ctx)
	if err != nil {
		return nil, err
	}
	if len(attrs) > 0 {
		out.Attrs = attrs
	}
	return out, nil
}

func lowerValue(n xparse.SyntaxNode[Term, Rule], ctx *Type) (*Value, error) {
	if rule, ok := n.Rule(); !ok || rule != RuleValue || n.ChildCount() != 1 {
		return nil, bad("value")
	}
	ch := n.Child(0)
	if rule, ok := ch.Rule(); ok {
		switch rule {
		case RuleGetElementPointer:
			return lowerConstantGetElementPointer(ch)
		case RuleConstantExpression:
			return &Value{Kind: ValExpr, Body: ch.Text()}, nil
		case RuleAggregate:
			return lowerAggregate(ch)
		case RuleMetadataNode:
			return &Value{Kind: ValMeta, Body: ch.Text()}, nil
		default:
			return nil, bad("value")
		}
	}
	switch ch.Name() {
	case TermLocalReference.String():
		name, err := sigilName(ch.Text())
		if err != nil {
			return nil, err
		}
		return Local(name), nil
	case TermGlobalReference.String():
		name, err := sigilName(ch.Text())
		if err != nil {
			return nil, err
		}
		return GlobalRef(name), nil
	case TermNumber.String():
		return numberValue(ch.Text(), ctx)
	case TermCString.String():
		text, err := decodeCString(ch.Text())
		if err != nil {
			return nil, err
		}
		return CString(text), nil
	case "true":
		return True(), nil
	case "false":
		return False(), nil
	case "null":
		return Null(ctx), nil
	case "undef":
		return Undef(ctx), nil
	case "poison":
		return Poison(ctx), nil
	case "zeroinitializer":
		return Zero(ctx), nil
	default:
		return nil, bad("value")
	}
}

func lowerConstantGetElementPointer(n xparse.SyntaxNode[Term, Rule]) (*Value, error) {
	v := &Value{Kind: ValGEP}
	r := &reader{n: n}
	if !r.eat("getelementptr") {
		return nil, bad("getelementptr")
	}
	for {
		switch r.name() {
		case "inbounds":
			v.InBounds = true
			r.pop()
		case "nuw", "nsw", "nusw", "inrange":
			v.Flags = append(v.Flags, r.pop().Text())
		default:
			goto body
		}
	}
body:
	if !r.eat("(") {
		return nil, bad("getelementptr")
	}
	src, err := lowerTypeNode(r.pop())
	if err != nil {
		return nil, err
	}
	v.SourceType = src
	if !r.eat(",") {
		return nil, bad("getelementptr")
	}
	base, err := lowerOperand(r.pop())
	if err != nil {
		return nil, err
	}
	v.PointerType = base.Type
	v.Base = base.Value
	for r.eat(",") {
		idx, err := lowerOperand(r.pop())
		if err != nil {
			return nil, err
		}
		v.Indices = append(v.Indices, idx)
	}
	if !r.eat(")") || r.ok() {
		return nil, bad("getelementptr")
	}
	return v, nil
}

func lowerAggregate(n xparse.SyntaxNode[Term, Rule]) (*Value, error) {
	switch n.Child(0).Name() {
	case "{":
		elems, err := aggregateElements(n)
		if err != nil {
			return nil, err
		}
		return &Value{Kind: ValStruct, Elems: elems}, nil
	case "[":
		elems, err := aggregateElements(n)
		if err != nil {
			return nil, err
		}
		return &Value{Kind: ValArray, Elems: elems}, nil
	case "<":
		if n.ChildCount() > 1 && n.Child(1).Name() == "{" {
			elems, err := aggregateElements(n)
			if err != nil {
				return nil, err
			}
			return &Value{Kind: ValPacked, Elems: elems}, nil
		}
		elems, err := aggregateElements(n)
		if err != nil {
			return nil, err
		}
		return &Value{Kind: ValVector, Elems: elems}, nil
	default:
		return nil, bad("aggregate")
	}
}

func aggregateElements(n xparse.SyntaxNode[Term, Rule]) ([]Operand, error) {
	var elems []Operand
	for i := 0; i < n.ChildCount(); i++ {
		ch := n.Child(i)
		rule, ok := ch.Rule()
		if !ok {
			continue
		}
		if rule != RuleOperand && rule != RuleOperandWithoutStruct {
			continue
		}
		op, err := lowerOperand(ch)
		if err != nil {
			return nil, err
		}
		elems = append(elems, op)
	}
	return elems, nil
}

func numberValue(text string, ctx *Type) (*Value, error) {
	floatTok := isFloatText(text)
	if ctx != nil && ctx.Kind == KindFloat {
		return FloatLit(ctx.Float, text), nil
	}
	if floatTok {
		if ctx != nil && ctx.Kind == KindPrim {
			return &Value{Kind: ValFloat, Text: text, Type: ctx}, nil
		}
		if ctx != nil && ctx.Kind != KindFloat {
			return nil, bad("integer")
		}
		return FloatLit(FloatDouble, text), nil
	}
	bits := 0
	if ctx != nil && ctx.Kind == KindInt {
		bits = ctx.Bits
	}
	if bits == 0 {
		return nil, bad("integer")
	}
	return Int(bits, text), nil
}

func integer(text string) (int64, error) {
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, bad("integer")
	}
	return n, nil
}

func sigilName(text string) (string, error) {
	if len(text) < 2 {
		return "", bad("name")
	}
	body := text[1:]
	if body[0] == '"' {
		name, ok := decodeString(body)
		if !ok {
			return "", bad("name")
		}
		return name, nil
	}
	return body, nil
}

func comdatName(text string) (string, error) {
	if !strings.HasPrefix(text, "$") {
		return "", bad("comdat")
	}
	body := strings.TrimSpace(text[1:])
	if strings.HasPrefix(body, "\"") {
		name, ok := decodeString(body)
		if !ok {
			return "", bad("comdat")
		}
		return name, nil
	}
	return body, nil
}

func decodeCString(text string) (string, error) {
	if len(text) < 2 || text[0] != 'c' {
		return "", bad("string")
	}
	out, ok := decodeString(text[1:])
	if !ok {
		return "", bad("string")
	}
	return out, nil
}

func decodeString(text string) (string, bool) {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return "", false
	}
	var b strings.Builder
	for i := 1; i < len(text)-1; i++ {
		if text[i] != '\\' {
			b.WriteByte(text[i])
			continue
		}
		i++
		if i >= len(text)-1 {
			return "", false
		}
		switch text[i] {
		case '\\', '"':
			b.WriteByte(text[i])
		default:
			if i+1 >= len(text)-1 || unhex(text[i]) < 0 || unhex(text[i+1]) < 0 {
				return "", false
			}
			b.WriteByte(byte(unhex(text[i])<<4 | unhex(text[i+1])))
			i++
		}
	}
	return b.String(), true
}
