package llvm

import (
	"fmt"
	"strconv"
	"strings"
)

type parser struct {
	lx      *lexer
	cur     token
	nxt     token
	far     token
	lastEnd int
}

func parse(src string) (*File, error) {
	p := &parser{lx: newLexer(src)}
	p.nxt = p.lx.next()
	p.far = p.lx.next()
	p.bump()
	file, err := p.parseFile()
	if err != nil {
		return nil, err
	}
	if p.lx.err != nil {
		return nil, p.lx.err
	}
	return file, nil
}

func (p *parser) bump() {
	if p.cur.end > p.lastEnd {
		p.lastEnd = p.cur.end
	}
	p.cur = p.nxt
	p.nxt = p.far
	p.far = p.lx.next()
}

func (p *parser) syntax(what string) error {
	return fmt.Errorf("%w: %d:%d: %s near %q", ErrSyntax, p.cur.line, p.cur.col, what, p.cur.text)
}

func (p *parser) unsupported(what string) error {
	return fmt.Errorf("%w: %d:%d: %s", ErrUnsupported, p.cur.line, p.cur.col, what)
}

func (p *parser) punct(c byte) bool {
	return p.cur.kind == tokPunct && p.cur.text == string(c)
}

func (p *parser) acceptPunct(c byte) bool {
	if p.punct(c) {
		p.bump()
		return true
	}
	return false
}

func (p *parser) expectPunct(c byte) error {
	if !p.acceptPunct(c) {
		return p.syntax(string(c))
	}
	return nil
}

func (p *parser) ident(s string) bool {
	return p.cur.kind == tokIdent && p.cur.text == s
}

func (p *parser) acceptIdent(s string) bool {
	if p.ident(s) {
		p.bump()
		return true
	}
	return false
}

func (p *parser) expectIdent(s string) error {
	if !p.acceptIdent(s) {
		return p.syntax(s)
	}
	return nil
}

func (p *parser) atLabel() bool {
	if p.nxt.kind != tokPunct || p.nxt.text != ":" {
		return false
	}
	return p.cur.kind == tokIdent || p.cur.kind == tokNumber
}

func (p *parser) atTop() bool {
	switch p.cur.kind {
	case tokEOF, tokGlobal, tokLocal:
		return true
	case tokPunct:
		return p.cur.text == "!"
	case tokIdent:
		switch p.cur.text {
		case "define", "declare", "attributes", "source_filename", "target", "module":
			return true
		}
	}
	return false
}

func (p *parser) parseFile() (*File, error) {
	if p.lx.err != nil {
		return nil, p.lx.err
	}
	file := New()
	for p.cur.kind != tokEOF {
		if p.lx.err != nil {
			return nil, p.lx.err
		}
		switch {
		case p.acceptIdent("source_filename"):
			if err := p.expectPunct('='); err != nil {
				return nil, err
			}
			text, err := p.expectString()
			if err != nil {
				return nil, err
			}
			file.SourceFilename = text
		case p.acceptIdent("target"):
			switch {
			case p.acceptIdent("datalayout"):
				if err := p.expectPunct('='); err != nil {
					return nil, err
				}
				text, err := p.expectString()
				if err != nil {
					return nil, err
				}
				file.DataLayout = text
			case p.acceptIdent("triple"):
				if err := p.expectPunct('='); err != nil {
					return nil, err
				}
				text, err := p.expectString()
				if err != nil {
					return nil, err
				}
				file.TargetTriple = text
			default:
				return nil, p.unsupported("target")
			}
		case p.cur.kind == tokLocal:
			if err := p.parseTypeDef(file); err != nil {
				return nil, err
			}
		case p.cur.kind == tokGlobal:
			if err := p.parseGlobal(file); err != nil {
				return nil, err
			}
		case p.ident("declare") || p.ident("define"):
			if err := p.parseFunc(file); err != nil {
				return nil, err
			}
		case p.punct('!'):
			if err := p.parseMetaDef(file); err != nil {
				return nil, err
			}
		case p.ident("attributes"):
			if err := p.parseAttrGroup(file); err != nil {
				return nil, err
			}
		case p.ident("module"):
			if err := p.parseModuleAsm(file); err != nil {
				return nil, err
			}
		case p.atComdat():
			if err := p.parseComdat(file); err != nil {
				return nil, err
			}
		default:
			return nil, p.syntax("module")
		}
	}
	return file, nil
}

func (p *parser) expectString() (string, error) {
	if p.cur.kind != tokString {
		return "", p.syntax("string")
	}
	text := p.cur.text
	p.bump()
	return text, nil
}

func (p *parser) parseTypeDef(file *File) error {
	def := &TypeDef{Name: p.cur.text}
	p.bump()
	if err := p.expectPunct('='); err != nil {
		return err
	}
	if err := p.expectIdent("type"); err != nil {
		return err
	}
	if p.acceptIdent("opaque") {
		def.IsOpaque = true
	} else {
		body, err := p.parseType()
		if err != nil {
			return err
		}
		def.Body = body
	}
	file.Types = append(file.Types, def)
	return nil
}

func (p *parser) parseGlobal(file *File) error {
	g := &Global{Name: p.cur.text}
	p.bump()
	if err := p.expectPunct('='); err != nil {
		return err
	}
	for {
		if p.cur.kind != tokIdent {
			return p.syntax("global")
		}
		switch p.cur.text {
		case "global", "constant", "alias", "ifunc":
			g.IsConstant = p.cur.text == "constant"
			g.Alias = p.cur.text == "alias"
			g.IFunc = p.cur.text == "ifunc"
			p.bump()
		default:
			if link, err := ParseLinkage(p.cur.text); err == nil {
				if g.Linkage != "" {
					return p.syntax("linkage")
				}
				g.Linkage = link
				p.bump()
				continue
			}
			switch p.cur.text {
			case "dso_local":
				g.DSOLocal = true
				p.bump()
				continue
			case "unnamed_addr":
				g.Unnamed = UnnamedAddr
				p.bump()
				continue
			case "local_unnamed_addr":
				g.Unnamed = UnnamedLocal
				p.bump()
				continue
			}
			word, err := p.takeWord()
			if err != nil {
				return err
			}
			g.Prefix = append(g.Prefix, word)
			continue
		}
		break
	}
	if g.Alias || g.IFunc {
		typ, err := p.parseType()
		if err != nil {
			return err
		}
		g.ValueType = typ
		if err := p.expectPunct(','); err != nil {
			return err
		}
		aliaseeType, err := p.parseType()
		if err != nil {
			return err
		}
		aliasee, err := p.parseTypedValue(aliaseeType)
		if err != nil {
			return err
		}
		g.Aliasee = &Operand{Type: aliaseeType, Value: aliasee}
	} else {
		typ, err := p.parseType()
		if err != nil {
			return err
		}
		g.ValueType = typ
		if p.valueStart() {
			init, err := p.parseTypedValue(typ)
			if err != nil {
				return err
			}
			g.Initializer = init
		}
	}
	if err := p.takeGlobalTrailer(g); err != nil {
		return err
	}
	file.Globals = append(file.Globals, g)
	return nil
}

func (p *parser) parseFunc(file *File) error {
	defined := p.ident("define")
	p.bump()
	fn := &Func{Defined: defined}
	for p.cur.kind == tokIdent && !p.typeStart() {
		if link, err := ParseLinkage(p.cur.text); err == nil {
			if fn.Linkage != "" {
				return p.syntax("linkage")
			}
			fn.Linkage = link
			p.bump()
			continue
		}
		switch p.cur.text {
		case "dso_local":
			fn.DSOLocal = true
			p.bump()
			continue
		case "unnamed_addr":
			fn.Unnamed = UnnamedAddr
			p.bump()
			continue
		case "local_unnamed_addr":
			fn.Unnamed = UnnamedLocal
			p.bump()
			continue
		}
		if isConv(p.cur.text) {
			conv, err := p.takeConv()
			if err != nil {
				return err
			}
			fn.Conv = conv
			continue
		}
		if isAttr(p.cur.text) {
			attrs, err := p.takeAttrs()
			if err != nil {
				return err
			}
			fn.ResultAttrs = append(fn.ResultAttrs, attrs...)
			continue
		}
		word, err := p.takeWord()
		if err != nil {
			return err
		}
		fn.Prefix = append(fn.Prefix, word)
	}
	ret, err := p.parseType()
	if err != nil {
		return err
	}
	fn.Result = ret
	if p.cur.kind != tokGlobal {
		return p.syntax("function name")
	}
	fn.Name = p.cur.text
	p.bump()
	params, variadic, err := p.parseParams()
	if err != nil {
		return err
	}
	fn.Parameters = params
	fn.Variadic = variadic
	if err := p.parseFuncSuffix(fn); err != nil {
		return err
	}
	if defined {
		if err := p.expectPunct('{'); err != nil {
			return err
		}
		if err := p.parseBody(fn); err != nil {
			return err
		}
		if err := p.expectPunct('}'); err != nil {
			return err
		}
	}
	file.Funcs = append(file.Funcs, fn)
	return nil
}

func (p *parser) parseParams() ([]*Param, bool, error) {
	if err := p.expectPunct('('); err != nil {
		return nil, false, err
	}
	if p.acceptPunct(')') {
		return nil, false, nil
	}
	var params []*Param
	variadic := false
	for {
		if p.cur.kind == tokDots {
			variadic = true
			p.bump()
			break
		}
		typ, err := p.parseType()
		if err != nil {
			return nil, false, err
		}
		attrs, err := p.takeAttrs()
		if err != nil {
			return nil, false, err
		}
		name := ""
		if p.cur.kind == tokLocal {
			name = p.cur.text
			p.bump()
		}
		params = append(params, &Param{Name: name, Type: typ, Attrs: attrs})
		if !p.acceptPunct(',') {
			break
		}
		if p.cur.kind == tokDots {
			variadic = true
			p.bump()
			break
		}
	}
	if err := p.expectPunct(')'); err != nil {
		return nil, false, err
	}
	return params, variadic, nil
}

func (p *parser) parseBody(fn *Func) error {
	for !p.punct('}') && p.cur.kind != tokEOF {
		if p.atLabel() {
			block := &Block{Name: p.cur.text}
			p.bump()
			p.bump()
			fn.Blocks = append(fn.Blocks, block)
			continue
		}
		if len(fn.Blocks) == 0 {
			fn.Blocks = append(fn.Blocks, &Block{})
		}
		in, err := p.parseInst()
		if err != nil {
			return err
		}
		last := fn.Blocks[len(fn.Blocks)-1]
		last.Instructions = append(last.Instructions, in)
	}
	return nil
}

func (p *parser) parseInst() (*Inst, error) {
	in := &Inst{}
	if p.cur.kind == tokLocal {
		in.Name = p.cur.text
		p.bump()
		if err := p.expectPunct('='); err != nil {
			return nil, err
		}
	}
	if p.cur.kind != tokIdent {
		return nil, p.syntax("instruction")
	}
	op := p.cur.text
	p.bump()
	switch op {
	case "tail", "musttail", "notail":
		in.Tail = Tail(op)
		if err := p.expectIdent("call"); err != nil {
			return nil, err
		}
		if err := p.parseCall(in); err != nil {
			return nil, err
		}
	case "alloca":
		if err := p.parseAlloca(in); err != nil {
			return nil, err
		}
	case "load":
		if err := p.parseLoad(in); err != nil {
			return nil, err
		}
	case "store":
		if err := p.parseStore(in); err != nil {
			return nil, err
		}
	case "getelementptr":
		if err := p.parseGEP(in); err != nil {
			return nil, err
		}
	case "icmp":
		if err := p.parseICmp(in); err != nil {
			return nil, err
		}
	case "fcmp":
		if err := p.parseFCmp(in); err != nil {
			return nil, err
		}
	case "phi":
		if err := p.parsePhi(in); err != nil {
			return nil, err
		}
	case "extractvalue":
		if err := p.parseExtract(in); err != nil {
			return nil, err
		}
	case "insertvalue":
		if err := p.parseInsert(in); err != nil {
			return nil, err
		}
	case "call":
		if err := p.parseCall(in); err != nil {
			return nil, err
		}
	case "br":
		if err := p.parseBr(in); err != nil {
			return nil, err
		}
	case "ret":
		if err := p.parseRet(in); err != nil {
			return nil, err
		}
	case "unreachable":
		in.Kind = InstUnreachable
		if err := p.skipTrailer(in, false); err != nil {
			return nil, err
		}
	case "fneg":
		if err := p.parseFNeg(in); err != nil {
			return nil, err
		}
	case "freeze":
		if err := p.parseFreeze(in); err != nil {
			return nil, err
		}
	case "select":
		if err := p.parseSelect(in); err != nil {
			return nil, err
		}
	case "extractelement":
		if err := p.parseExtractElt(in); err != nil {
			return nil, err
		}
	case "insertelement":
		if err := p.parseInsertElt(in); err != nil {
			return nil, err
		}
	case "shufflevector":
		if err := p.parseShuffle(in); err != nil {
			return nil, err
		}
	case "atomicrmw":
		if err := p.parseAtomicRMW(in); err != nil {
			return nil, err
		}
	case "cmpxchg":
		if err := p.parseCmpXchg(in); err != nil {
			return nil, err
		}
	case "fence":
		if err := p.parseFence(in); err != nil {
			return nil, err
		}
	case "switch":
		if err := p.parseSwitch(in); err != nil {
			return nil, err
		}
	case "invoke":
		if err := p.parseInvoke(in); err != nil {
			return nil, err
		}
	case "callbr":
		if err := p.parseCallBr(in); err != nil {
			return nil, err
		}
	case "resume":
		if err := p.parseResume(in); err != nil {
			return nil, err
		}
	case "landingpad":
		if err := p.parseLanding(in); err != nil {
			return nil, err
		}
	case "indirectbr":
		if err := p.parseIndirectBr(in); err != nil {
			return nil, err
		}
	case "va_arg":
		if err := p.parseVAArg(in); err != nil {
			return nil, err
		}
	default:
		if cast, err := ParseCast(op); err == nil {
			if err := p.parseCast(in, cast); err != nil {
				return nil, err
			}
			break
		}
		if bin, err := ParseOp(op); err == nil {
			if err := p.parseBin(in, bin); err != nil {
				return nil, err
			}
			break
		}
		return nil, p.unsupported(op)
	}
	return in, nil
}

func (p *parser) parseAlloca(in *Inst) error {
	in.Kind = InstAlloca
	if p.acceptIdent("inalloca") {
		in.Flags = append(in.Flags, "inalloca")
	}
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	if p.punct(',') && !(p.nxt.kind == tokIdent && p.nxt.text == "align") && !(p.nxt.kind == tokPunct && p.nxt.text == "!") {
		p.bump()
		countType, err := p.parseType()
		if err != nil {
			return err
		}
		count, err := p.parseTypedValue(countType)
		if err != nil {
			return err
		}
		in.Count = &Operand{Type: countType, Value: count}
	}
	return p.skipTrailer(in, true)
}

func (p *parser) parseLoad(in *Inst) error {
	in.Kind = InstLoad
	if p.acceptIdent("atomic") {
		in.Atomic = true
	}
	if p.acceptIdent("volatile") {
		in.IsVolatile = true
	}
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	if err := p.expectPunct(','); err != nil {
		return err
	}
	pointer, err := p.parseType()
	if err != nil {
		return err
	}
	in.Pointer = pointer
	value, err := p.parseTypedValue(pointer)
	if err != nil {
		return err
	}
	in.Left = value
	if err := p.takeAtomicTail(in); err != nil {
		return err
	}
	return p.skipTrailer(in, true)
}

func (p *parser) parseStore(in *Inst) error {
	in.Kind = InstStore
	if p.acceptIdent("atomic") {
		in.Atomic = true
	}
	if p.acceptIdent("volatile") {
		in.IsVolatile = true
	}
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	value, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = value
	if err := p.expectPunct(','); err != nil {
		return err
	}
	pointer, err := p.parseType()
	if err != nil {
		return err
	}
	in.Pointer = pointer
	dest, err := p.parseTypedValue(pointer)
	if err != nil {
		return err
	}
	in.Right = dest
	if err := p.takeAtomicTail(in); err != nil {
		return err
	}
	return p.skipTrailer(in, true)
}

func (p *parser) parseGEP(in *Inst) error {
	in.Kind = InstGEP
	p.takeGEPFlags(&in.Flags, &in.InBounds, &in.NoUnsignedWrap, &in.NoSignedWrap)
	source, err := p.parseType()
	if err != nil {
		return err
	}
	in.Source = source
	if err := p.expectPunct(','); err != nil {
		return err
	}
	pointer, err := p.parseType()
	if err != nil {
		return err
	}
	in.Pointer = pointer
	base, err := p.parseTypedValue(pointer)
	if err != nil {
		return err
	}
	in.Left = base
	for p.punct(',') {
		if p.nxt.kind == tokIdent && p.nxt.text == "align" {
			break
		}
		if p.nxt.kind == tokPunct && p.nxt.text == "!" {
			break
		}
		p.bump()
		idxType, err := p.parseType()
		if err != nil {
			return err
		}
		idx, err := p.parseTypedValue(idxType)
		if err != nil {
			return err
		}
		in.Indices = append(in.Indices, Operand{Type: idxType, Value: idx})
	}
	return p.skipTrailer(in, true)
}

func (p *parser) parseBin(in *Inst, op Op) error {
	in.Kind = InstBin
	in.Op = op
	for p.cur.kind == tokIdent && !p.typeStart() {
		switch p.cur.text {
		case "nsw":
			in.NoSignedWrap = true
		case "nuw":
			in.NoUnsignedWrap = true
		case "exact":
			in.ExactDiv = true
		case "disjoint":
			in.Flags = append(in.Flags, "disjoint")
		default:
			if !isFastMath(p.cur.text) {
				return p.unsupported(p.cur.text)
			}
			in.Flags = append(in.Flags, p.cur.text)
		}
		p.bump()
	}
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	left, err := p.parseValue(typ)
	if err != nil {
		return err
	}
	in.Left = left
	if err := p.expectPunct(','); err != nil {
		return err
	}
	right, err := p.parseValue(typ)
	if err != nil {
		return err
	}
	in.Right = right
	return p.skipTrailer(in, false)
}

func (p *parser) parseICmp(in *Inst) error {
	in.Kind = InstICmp
	for p.cur.kind == tokIdent {
		if _, err := ParseIPred(p.cur.text); err == nil {
			break
		}
		in.Flags = append(in.Flags, p.cur.text)
		p.bump()
	}
	if p.cur.kind != tokIdent {
		return p.syntax("predicate")
	}
	pred, err := ParseIPred(p.cur.text)
	if err != nil {
		return p.unsupported(p.cur.text)
	}
	in.IntegerPred = pred
	p.bump()
	return p.parseCmpValues(in)
}

func (p *parser) parseFCmp(in *Inst) error {
	in.Kind = InstFCmp
	in.Flags = p.takeFastMath()
	if p.cur.kind != tokIdent {
		return p.syntax("predicate")
	}
	pred, err := ParseFPred(p.cur.text)
	if err != nil {
		return p.unsupported(p.cur.text)
	}
	in.FloatPred = pred
	p.bump()
	return p.parseCmpValues(in)
}

func isFPred(text string) bool {
	_, err := ParseFPred(text)
	return err == nil
}

func (p *parser) parseCmpValues(in *Inst) error {
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	left, err := p.parseValue(typ)
	if err != nil {
		return err
	}
	in.Left = left
	if err := p.expectPunct(','); err != nil {
		return err
	}
	right, err := p.parseValue(typ)
	if err != nil {
		return err
	}
	in.Right = right
	return p.skipTrailer(in, false)
}

func (p *parser) parseCast(in *Inst, kind Cast) error {
	in.Kind = InstCast
	in.Cast = kind
	for p.cur.kind == tokIdent && !p.typeStart() {
		switch p.cur.text {
		case "nuw":
			in.NoUnsignedWrap = true
		case "nsw":
			in.NoSignedWrap = true
		case "nneg":
			in.Flags = append(in.Flags, "nneg")
		default:
			if !isFastMath(p.cur.text) {
				return p.unsupported(p.cur.text)
			}
			in.Flags = append(in.Flags, p.cur.text)
		}
		p.bump()
	}
	from, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = from
	value, err := p.parseTypedValue(from)
	if err != nil {
		return err
	}
	in.Left = value
	if err := p.expectIdent("to"); err != nil {
		return err
	}
	dest, err := p.parseType()
	if err != nil {
		return err
	}
	in.Dest = dest
	return p.skipTrailer(in, false)
}

func (p *parser) parsePhi(in *Inst) error {
	in.Kind = InstPhi
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	for {
		if err := p.expectPunct('['); err != nil {
			return err
		}
		value, err := p.parseValue(typ)
		if err != nil {
			return err
		}
		if err := p.expectPunct(','); err != nil {
			return err
		}
		if p.cur.kind != tokLocal {
			return p.syntax("phi label")
		}
		in.Incoming = append(in.Incoming, Incoming{Value: value, Block: p.cur.text})
		p.bump()
		if err := p.expectPunct(']'); err != nil {
			return err
		}
		if !p.acceptPunct(',') {
			break
		}
	}
	return p.skipTrailer(in, false)
}

func (p *parser) parseExtract(in *Inst) error {
	in.Kind = InstExtract
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	value, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = value
	path, err := p.parsePath()
	if err != nil {
		return err
	}
	in.Path = path
	return p.skipTrailer(in, false)
}

func (p *parser) parseInsert(in *Inst) error {
	in.Kind = InstInsert
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	agg, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = agg
	if err := p.expectPunct(','); err != nil {
		return err
	}
	elemType, err := p.parseType()
	if err != nil {
		return err
	}
	elem, err := p.parseTypedValue(elemType)
	if err != nil {
		return err
	}
	in.Inserted = &Operand{Type: elemType, Value: elem}
	path, err := p.parsePath()
	if err != nil {
		return err
	}
	in.Path = path
	return p.skipTrailer(in, false)
}

func (p *parser) parsePath() ([]int, error) {
	var path []int
	for p.punct(',') && !(p.nxt.kind == tokPunct && p.nxt.text == "!") {
		p.bump()
		n, err := p.integer()
		if err != nil {
			return nil, err
		}
		path = append(path, int(n))
	}
	if len(path) == 0 {
		return nil, p.syntax("index")
	}
	return path, nil
}

func (p *parser) parseCall(in *Inst) error {
	in.Kind = InstCall
	in.Flags = p.takeFastMath()
	if p.cur.kind == tokIdent && isConv(p.cur.text) {
		conv, err := p.takeConv()
		if err != nil {
			return err
		}
		in.Conv = conv
	}
	retAttrs, err := p.takeAttrs()
	if err != nil {
		return err
	}
	in.ResultAttrs = retAttrs
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	if typ.Kind == KindFunc {
		in.Signature = typ
		in.Type = typ.Ret
	} else {
		in.Type = typ
	}
	callee, err := p.parseTypedValue(nil)
	if err != nil {
		return err
	}
	in.Callee = callee
	if err := p.expectPunct('('); err != nil {
		return err
	}
	if !p.acceptPunct(')') {
		for {
			argType, err := p.parseType()
			if err != nil {
				return err
			}
			arg, err := p.parseTypedValue(argType)
			if err != nil {
				return err
			}
			in.Args = append(in.Args, Operand{Type: argType, Value: arg})
			if !p.acceptPunct(',') {
				break
			}
		}
		if err := p.expectPunct(')'); err != nil {
			return err
		}
	}
	if err := p.takeCallSuffix(in); err != nil {
		return err
	}
	return p.skipTrailer(in, false)
}

func (p *parser) parseBr(in *Inst) error {
	in.Kind = InstBr
	if p.acceptIdent("label") {
		if p.cur.kind != tokLocal {
			return p.syntax("label")
		}
		in.Target = p.cur.text
		p.bump()
		return p.skipTrailer(in, false)
	}
	if _, err := p.parseType(); err != nil {
		return err
	}
	cond, err := p.parseTypedValue(IntType(1))
	if err != nil {
		return err
	}
	in.Condition = cond
	if err := p.expectPunct(','); err != nil {
		return err
	}
	if err := p.expectIdent("label"); err != nil {
		return err
	}
	if p.cur.kind != tokLocal {
		return p.syntax("label")
	}
	in.TrueLabel = p.cur.text
	p.bump()
	if err := p.expectPunct(','); err != nil {
		return err
	}
	if err := p.expectIdent("label"); err != nil {
		return err
	}
	if p.cur.kind != tokLocal {
		return p.syntax("label")
	}
	in.FalseLabel = p.cur.text
	p.bump()
	return p.skipTrailer(in, false)
}

func (p *parser) parseRet(in *Inst) error {
	in.Kind = InstRet
	if p.acceptIdent("void") {
		in.Type = Void()
		return p.skipTrailer(in, false)
	}
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	value, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = value
	return p.skipTrailer(in, false)
}

func (p *parser) skipTrailer(in *Inst, align bool) error {
	for p.acceptPunct(',') {
		if align && p.acceptIdent("align") {
			n, err := p.integer()
			if err != nil {
				return err
			}
			in.Alignment = int(n)
			continue
		}
		if p.punct('!') {
			att, err := p.takeAttach()
			if err != nil {
				return err
			}
			in.Attach = append(in.Attach, att)
			continue
		}
		return p.unsupported("instruction trailer")
	}
	return nil
}

func (p *parser) parseType() (*Type, error) {
	primary, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	typed, err := p.parseStars(primary)
	if err != nil {
		return nil, err
	}
	if p.acceptPunct('(') {
		params, variadic, err := p.parseTypeList()
		if err != nil {
			return nil, err
		}
		typed = &Type{Kind: KindFunc, Ret: typed, Fields: params, Variadic: variadic}
		typed, err = p.parseStars(typed)
		if err != nil {
			return nil, err
		}
	}
	return typed, nil
}

func (p *parser) parsePrimary() (*Type, error) {
	switch p.cur.kind {
	case tokLocal:
		name := p.cur.text
		p.bump()
		return Named(name), nil
	case tokPunct:
		switch p.cur.text {
		case "[":
			p.bump()
			n, err := p.integer()
			if err != nil {
				return nil, err
			}
			if err := p.expectIdent("x"); err != nil {
				return nil, err
			}
			elem, err := p.parseType()
			if err != nil {
				return nil, err
			}
			if err := p.expectPunct(']'); err != nil {
				return nil, err
			}
			return Array(n, elem), nil
		case "{":
			p.bump()
			fields, err := p.parseTypeSeq('}')
			if err != nil {
				return nil, err
			}
			return Struct(fields...), nil
		case "<":
			p.bump()
			if p.acceptPunct('{') {
				fields, err := p.parseTypeSeq('}')
				if err != nil {
					return nil, err
				}
				if err := p.expectPunct('>'); err != nil {
					return nil, err
				}
				return PackedStruct(fields...), nil
			}
			scalable := false
			if p.acceptIdent("vscale") {
				scalable = true
				if err := p.expectIdent("x"); err != nil {
					return nil, err
				}
			}
			n, err := p.integer()
			if err != nil {
				return nil, err
			}
			if err := p.expectIdent("x"); err != nil {
				return nil, err
			}
			elem, err := p.parseType()
			if err != nil {
				return nil, err
			}
			if err := p.expectPunct('>'); err != nil {
				return nil, err
			}
			typ := Vector(n, elem)
			typ.Scalable = scalable
			return typ, nil
		default:
			return nil, p.syntax("type")
		}
	case tokIdent:
		text := p.cur.text
		switch text {
		case "void":
			p.bump()
			return Void(), nil
		case "half":
			p.bump()
			return FP(FloatHalf), nil
		case "float":
			p.bump()
			return FP(FloatFloat), nil
		case "double":
			p.bump()
			return FP(FloatDouble), nil
		case "ptr":
			p.bump()
			typ := Ptr()
			if p.ident("addrspace") {
				space, err := p.parseAddrSpace()
				if err != nil {
					return nil, err
				}
				typ.AddrSpace = space
			}
			return typ, nil
		case "metadata":
			p.bump()
			return &Type{Kind: KindMeta}, nil
		case "token":
			p.bump()
			return &Type{Kind: KindToken}, nil
		case "label":
			p.bump()
			return &Type{Kind: KindLabel}, nil
		case "bfloat", "x86_fp80", "fp128", "ppc_fp128", "x86_mmx", "x86_amx":
			p.bump()
			return &Type{Kind: KindPrim, Name: text}, nil
		}
		if bits, ok := intBits(text); ok {
			p.bump()
			return IntType(bits), nil
		}
		return nil, p.syntax("type")
	default:
		return nil, p.syntax("type")
	}
}

func (p *parser) parseStars(typ *Type) (*Type, error) {
	for {
		space := 0
		if p.ident("addrspace") {
			n, err := p.parseAddrSpace()
			if err != nil {
				return nil, err
			}
			space = n
			if !p.punct('*') {
				return nil, p.syntax("pointer")
			}
		}
		if !p.acceptPunct('*') {
			return typ, nil
		}
		typ = &Type{Kind: KindPtr, Elem: typ, AddrSpace: space}
	}
}

func (p *parser) parseAddrSpace() (int, error) {
	if err := p.expectIdent("addrspace"); err != nil {
		return 0, err
	}
	if err := p.expectPunct('('); err != nil {
		return 0, err
	}
	n, err := p.integer()
	if err != nil {
		return 0, err
	}
	if err := p.expectPunct(')'); err != nil {
		return 0, err
	}
	return int(n), nil
}

func (p *parser) parseTypeSeq(until byte) ([]*Type, error) {
	if p.acceptPunct(until) {
		return nil, nil
	}
	var fields []*Type
	for {
		typ, err := p.parseType()
		if err != nil {
			return nil, err
		}
		fields = append(fields, typ)
		if p.acceptPunct(',') {
			continue
		}
		break
	}
	if err := p.expectPunct(until); err != nil {
		return nil, err
	}
	return fields, nil
}

func (p *parser) parseTypeList() ([]*Type, bool, error) {
	if p.acceptPunct(')') {
		return nil, false, nil
	}
	var params []*Type
	variadic := false
	for {
		if p.cur.kind == tokDots {
			variadic = true
			p.bump()
			break
		}
		typ, err := p.parseType()
		if err != nil {
			return nil, false, err
		}
		params = append(params, typ)
		if !p.acceptPunct(',') {
			break
		}
		if p.cur.kind == tokDots {
			variadic = true
			p.bump()
			break
		}
	}
	if err := p.expectPunct(')'); err != nil {
		return nil, false, err
	}
	return params, variadic, nil
}

func (p *parser) parseValue(ctx *Type) (*Value, error) {
	switch p.cur.kind {
	case tokLocal:
		value := Local(p.cur.text)
		p.bump()
		return value, nil
	case tokGlobal:
		value := GlobalRef(p.cur.text)
		p.bump()
		return value, nil
	case tokNumber:
		text := p.cur.text
		p.bump()
		return p.numberValue(text, ctx)
	case tokCString:
		value := CString(p.cur.text)
		p.bump()
		return value, nil
	case tokIdent:
		switch p.cur.text {
		case "true":
			p.bump()
			return True(), nil
		case "false":
			p.bump()
			return False(), nil
		case "null":
			p.bump()
			return Null(ctx), nil
		case "undef":
			p.bump()
			return Undef(ctx), nil
		case "poison":
			p.bump()
			return Poison(ctx), nil
		case "zeroinitializer":
			p.bump()
			return Zero(ctx), nil
		case "getelementptr":
			p.bump()
			return p.parseConstGEP()
		default:
			if p.constExprStart() {
				return p.parseConstExpr()
			}
			return nil, p.syntax("value")
		}
	case tokPunct:
		if p.cur.text == "!" {
			return p.parseMetaValue()
		}
		switch p.cur.text {
		case "{":
			return p.parseAgg(ValStruct, '{', '}')
		case "[":
			return p.parseAgg(ValArray, '[', ']')
		case "<":
			if p.nxt.kind == tokPunct && p.nxt.text == "{" {
				p.bump()
				value, err := p.parseAgg(ValPacked, '{', '}')
				if err != nil {
					return nil, err
				}
				if err := p.expectPunct('>'); err != nil {
					return nil, err
				}
				return value, nil
			}
			return p.parseAgg(ValVector, '<', '>')
		default:
			return nil, p.syntax("value")
		}
	default:
		return nil, p.syntax("value")
	}
}

func (p *parser) parseConstGEP() (*Value, error) {
	value := &Value{Kind: ValGEP}
	p.takeGEPFlags(&value.Flags, &value.InBounds, nil, nil)
	if err := p.expectPunct('('); err != nil {
		return nil, err
	}
	source, err := p.parseType()
	if err != nil {
		return nil, err
	}
	value.SourceType = source
	if err := p.expectPunct(','); err != nil {
		return nil, err
	}
	pointer, err := p.parseType()
	if err != nil {
		return nil, err
	}
	value.PointerType = pointer
	base, err := p.parseTypedValue(pointer)
	if err != nil {
		return nil, err
	}
	value.Base = base
	for p.acceptPunct(',') {
		idxType, err := p.parseType()
		if err != nil {
			return nil, err
		}
		idx, err := p.parseTypedValue(idxType)
		if err != nil {
			return nil, err
		}
		value.Indices = append(value.Indices, Operand{Type: idxType, Value: idx})
	}
	if err := p.expectPunct(')'); err != nil {
		return nil, err
	}
	return value, nil
}

func (p *parser) parseAgg(kind ValueKind, open, close byte) (*Value, error) {
	if err := p.expectPunct(open); err != nil {
		return nil, err
	}
	var elems []Operand
	if !p.punct(close) {
		for {
			typ, err := p.parseType()
			if err != nil {
				return nil, err
			}
			value, err := p.parseTypedValue(typ)
			if err != nil {
				return nil, err
			}
			elems = append(elems, Operand{Type: typ, Value: value})
			if p.acceptPunct(',') {
				continue
			}
			break
		}
	}
	if err := p.expectPunct(close); err != nil {
		return nil, err
	}
	return &Value{Kind: kind, Elems: elems}, nil
}

func (p *parser) numberValue(text string, ctx *Type) (*Value, error) {
	floatTok := isFloatText(text)
	if ctx != nil && ctx.Kind == KindFloat {
		return FloatLit(ctx.Float, text), nil
	}
	if floatTok {
		if ctx != nil && ctx.Kind == KindPrim {
			return &Value{Kind: ValFloat, Text: text, Type: ctx}, nil
		}
		if ctx != nil && ctx.Kind != KindFloat {
			return nil, p.syntax("integer")
		}
		return FloatLit(FloatDouble, text), nil
	}
	bits := 0
	if ctx != nil && ctx.Kind == KindInt {
		bits = ctx.Bits
	}
	if bits == 0 {
		return nil, p.syntax("integer")
	}
	return Int(bits, text), nil
}

func (p *parser) integer() (int64, error) {
	if p.cur.kind != tokNumber {
		return 0, p.syntax("integer")
	}
	n, err := strconv.ParseInt(p.cur.text, 10, 64)
	if err != nil {
		return 0, p.syntax("integer")
	}
	p.bump()
	return n, nil
}

func (p *parser) valueStart() bool {
	// A sigil followed by '=' is the next definition, not an initializer.
	if (p.cur.kind == tokLocal || p.cur.kind == tokGlobal) && p.nxt.kind == tokPunct && p.nxt.text == "=" {
		return false
	}
	switch p.cur.kind {
	case tokNumber, tokLocal, tokGlobal, tokCString:
		return true
	case tokPunct:
		return p.cur.text == "{" || p.cur.text == "[" || p.cur.text == "<"
	case tokIdent:
		if p.constExprStart() {
			return true
		}
		switch p.cur.text {
		case "true", "false", "null", "undef", "poison", "zeroinitializer", "getelementptr":
			return true
		}
	}
	return false
}

func (p *parser) typeStart() bool {
	switch p.cur.kind {
	case tokLocal:
		return true
	case tokPunct:
		return p.cur.text == "[" || p.cur.text == "{" || p.cur.text == "<"
	case tokIdent:
		switch p.cur.text {
		case "void", "half", "float", "double", "ptr", "metadata", "token", "label", "bfloat", "x86_fp80", "fp128", "ppc_fp128", "x86_mmx", "x86_amx":
			return true
		}
		_, ok := intBits(p.cur.text)
		return ok
	default:
		return false
	}
}

func isAttr(name string) bool {
	switch name {
	case "noundef", "nonnull", "noalias", "nocapture", "readonly", "readnone", "writeonly",
		"signext", "zeroext", "inreg", "returned", "align", "alignstack", "nneg", "nofpclass",
		"dereferenceable", "dereferenceable_or_null", "sret", "byval", "byref", "preallocated",
		"inalloca", "elementtype", "immarg", "nofree", "nest", "swiftself", "swifterror",
		"swiftasync", "captures", "writable", "dead_on_unwind", "dead_on_return", "range",
		"allocalign", "allocptr", "allocsize", "initializes":
		return true
	default:
		return false
	}
}

func (p *parser) skipDelim(open, close byte) error {
	if !p.punct(open) {
		return p.syntax("delimiter")
	}
	depth := 0
	for {
		if p.cur.kind == tokEOF {
			return p.syntax("delimiter")
		}
		if p.punct(open) {
			depth++
		}
		if p.punct(close) {
			depth--
			if depth == 0 {
				p.bump()
				return nil
			}
		}
		p.bump()
	}
}

func intBits(text string) (int, bool) {
	if len(text) < 2 || text[0] != 'i' {
		return 0, false
	}
	for _, c := range text[1:] {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(text[1:])
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func isFloatText(text string) bool {
	if strings.ContainsAny(text, ".pP") {
		return true
	}
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "0x") || strings.HasPrefix(lower, "-0x") {
		return true
	}
	return strings.ContainsAny(lower, "e")
}
