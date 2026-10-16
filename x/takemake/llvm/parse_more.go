package llvm

import "strings"

func (p *parser) spanFrom(from int) string {
	end := p.lastEnd
	if from < 0 {
		from = 0
	}
	if end > len(p.lx.s) {
		end = len(p.lx.s)
	}
	if from > end {
		return ""
	}
	return p.lx.s[from:end]
}

func (p *parser) atComdat() bool {
	return p.cur.kind == tokIdent && strings.HasPrefix(p.cur.text, "$")
}

func (p *parser) atFuncAttach() bool {
	if !p.punct('!') {
		return false
	}
	return !(p.far.kind == tokPunct && p.far.text == "=")
}

func (p *parser) suffixDone() bool {
	if p.punct('{') || p.cur.kind == tokEOF {
		return true
	}
	if p.atFuncAttach() {
		return false
	}
	return p.atTop()
}

func (p *parser) takeAttrs() ([]string, error) {
	var attrs []string
	for p.cur.kind == tokIdent && !p.typeStart() && !p.attrStop() {
		one, err := p.takeOneAttr()
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, one)
	}
	return attrs, nil
}

// attrStop reports an identifier that ends an attribute list: a value,
// a calling convention, or a top-level keyword. Other identifiers are
// attributes, including ones this package does not name individually.
func (p *parser) attrStop() bool {
	text := p.cur.text
	switch text {
	case "true", "false", "null", "undef", "poison", "zeroinitializer", "getelementptr",
		"blockaddress", "dso_local", "unnamed_addr", "local_unnamed_addr",
		"global", "constant", "alias", "ifunc", "define", "declare", "attributes",
		"type", "opaque", "comdat", "module", "asm", "source_filename", "target",
		"icmp", "fcmp", "select", "freeze", "fneg", "extractvalue", "insertvalue",
		"extractelement", "insertelement", "shufflevector":
		return true
	}
	if isConv(text) {
		return true
	}
	if _, err := ParseLinkage(text); err == nil {
		return true
	}
	if _, err := ParseCast(text); err == nil {
		return true
	}
	if _, err := ParseOp(text); err == nil {
		return true
	}
	return false
}

func (p *parser) takeOneAttr() (string, error) {
	from := p.cur.start
	name := p.cur.text
	p.bump()
	if p.punct('(') {
		if err := p.skipDelim('(', ')'); err != nil {
			return "", err
		}
	} else if (name == "align" || name == "alignstack") && p.cur.kind == tokNumber {
		p.bump()
	}
	return p.spanFrom(from), nil
}

func (p *parser) parseTypedValue(typ *Type) (*Value, error) {
	attrs, err := p.takeAttrs()
	if err != nil {
		return nil, err
	}
	value, err := p.parseValue(typ)
	if err != nil {
		return nil, err
	}
	if len(attrs) > 0 {
		value.Attrs = attrs
	}
	return value, nil
}

func (p *parser) takeFastMath() []string {
	var flags []string
	for p.cur.kind == tokIdent && isFastMath(p.cur.text) {
		flags = append(flags, p.cur.text)
		p.bump()
	}
	return flags
}

func isFastMath(name string) bool {
	switch name {
	case "nnan", "ninf", "nsz", "arcp", "contract", "afn", "reassoc", "fast":
		return true
	default:
		return false
	}
}

func isConv(name string) bool {
	switch name {
	case "ccc", "fastcc", "coldcc", "ghccc", "cc", "webkit_jscc", "anyregcc",
		"preserve_mostcc", "preserve_allcc", "preserve_nonecc", "cxx_fast_tlscc",
		"tailcc", "swiftcc", "swifttailcc", "cfguard_checkcc", "nakedcc",
		"intel_ocl_bicc", "x86_stdcallcc", "x86_fastcallcc", "x86_thiscallcc",
		"x86_vectorcallcc", "x86_regcallcc", "arm_apcscc", "arm_aapcscc",
		"arm_aapcs_vfpcc", "aarch64_vector_pcs", "aarch64_sve_vector_pcs",
		"msp430_intrcc", "avr_intrcc", "avr_signalcc", "ptx_kernel", "ptx_device",
		"spir_func", "spir_kernel", "x86_64_sysvcc", "win64cc", "amdgpu_vs",
		"amdgpu_ls", "amdgpu_hs", "amdgpu_es", "amdgpu_gs", "amdgpu_ps",
		"amdgpu_cs", "amdgpu_kernel", "amdgpu_gfx", "hhvmcc", "hhvm_ccc":
		return true
	default:
		return false
	}
}

func (p *parser) takeConv() (string, error) {
	from := p.cur.start
	if p.cur.text == "cc" {
		p.bump()
		if _, err := p.integer(); err != nil {
			return "", err
		}
		return p.spanFrom(from), nil
	}
	text := p.cur.text
	p.bump()
	return text, nil
}

func (p *parser) takeWord() (string, error) {
	from := p.cur.start
	if p.cur.kind != tokIdent {
		return "", p.syntax("specifier")
	}
	p.bump()
	if p.punct('(') {
		if err := p.skipDelim('(', ')'); err != nil {
			return "", err
		}
	}
	return p.spanFrom(from), nil
}

func (p *parser) parseModuleAsm(file *File) error {
	if err := p.expectIdent("module"); err != nil {
		return err
	}
	if err := p.expectIdent("asm"); err != nil {
		return err
	}
	text, err := p.expectString()
	if err != nil {
		return err
	}
	file.Asms = append(file.Asms, text)
	return nil
}

func (p *parser) parseComdat(file *File) error {
	name, err := p.parseComdatName()
	if err != nil {
		return err
	}
	if err := p.expectPunct('='); err != nil {
		return err
	}
	if err := p.expectIdent("comdat"); err != nil {
		return err
	}
	if p.cur.kind != tokIdent {
		return p.syntax("comdat")
	}
	kind, err := ParseComdatKind(p.cur.text)
	if err != nil {
		return p.unsupported(p.cur.text)
	}
	p.bump()
	file.Comdats = append(file.Comdats, &Comdat{Name: name, Kind: kind})
	return nil
}

func (p *parser) parseComdatName() (string, error) {
	if p.cur.kind != tokIdent || !strings.HasPrefix(p.cur.text, "$") {
		return "", p.syntax("comdat")
	}
	if p.cur.text == "$" {
		p.bump()
		if p.cur.kind == tokString {
			name := p.cur.text
			p.bump()
			return name, nil
		}
		if p.cur.kind == tokIdent {
			name := p.cur.text
			p.bump()
			return name, nil
		}
		return "", p.syntax("comdat")
	}
	name := p.cur.text[1:]
	p.bump()
	return name, nil
}

func (p *parser) parseAttrGroup(file *File) error {
	if err := p.expectIdent("attributes"); err != nil {
		return err
	}
	if err := p.expectPunct('#'); err != nil {
		return err
	}
	if p.cur.kind != tokNumber {
		return p.syntax("attribute group")
	}
	id := p.cur.text
	p.bump()
	if err := p.expectPunct('='); err != nil {
		return err
	}
	from := p.cur.start
	if err := p.skipDelim('{', '}'); err != nil {
		return err
	}
	file.AttrGroups = append(file.AttrGroups, &AttrGroup{ID: id, Body: p.spanFrom(from)})
	return nil
}

func (p *parser) parseMetaDef(file *File) error {
	if err := p.expectPunct('!'); err != nil {
		return err
	}
	if p.cur.kind != tokIdent && p.cur.kind != tokNumber {
		return p.syntax("metadata")
	}
	name := p.cur.text
	p.bump()
	if err := p.expectPunct('='); err != nil {
		return err
	}
	rhs, err := p.takeMDRhs()
	if err != nil {
		return err
	}
	file.Metadata = append(file.Metadata, &MetaDef{Name: name, RHS: rhs})
	return nil
}

func (p *parser) takeMDRhs() (string, error) {
	from := p.cur.start
	if err := p.consumeMDValue(); err != nil {
		return "", err
	}
	return p.spanFrom(from), nil
}

func (p *parser) consumeMDValue() error {
	p.acceptIdent("distinct")
	if err := p.expectPunct('!'); err != nil {
		return err
	}
	switch p.cur.kind {
	case tokNumber, tokString:
		p.bump()
		return nil
	case tokIdent:
		p.bump()
		if p.punct('(') {
			return p.skipDelim('(', ')')
		}
		return nil
	case tokPunct:
		if p.cur.text == "{" {
			return p.skipDelim('{', '}')
		}
	}
	return p.syntax("metadata")
}

func (p *parser) takeAttach() (Attach, error) {
	if err := p.expectPunct('!'); err != nil {
		return Attach{}, err
	}
	if p.cur.kind != tokIdent && p.cur.kind != tokNumber {
		return Attach{}, p.syntax("metadata")
	}
	name := p.cur.text
	p.bump()
	body, err := p.takeMDRhs()
	if err != nil {
		return Attach{}, err
	}
	return Attach{Name: name, Body: body}, nil
}

func (p *parser) parseFuncSuffix(fn *Func) error {
	for !p.suffixDone() {
		switch {
		case p.atFuncAttach():
			att, err := p.takeAttach()
			if err != nil {
				return err
			}
			fn.Attach = append(fn.Attach, att)
		case p.acceptIdent("unnamed_addr"):
			fn.Unnamed = UnnamedAddr
		case p.acceptIdent("local_unnamed_addr"):
			fn.Unnamed = UnnamedLocal
		default:
			piece, err := p.takeSuffixPiece()
			if err != nil {
				return err
			}
			fn.Suffix = append(fn.Suffix, piece)
		}
	}
	return nil
}

func (p *parser) takeSuffixPiece() (string, error) {
	from := p.cur.start
	switch {
	case p.punct('#'):
		p.bump()
		if p.cur.kind != tokNumber {
			return "", p.syntax("attribute group")
		}
		p.bump()
	case p.ident("personality"), p.ident("prefix"), p.ident("prologue"):
		p.bump()
		typ, err := p.parseType()
		if err != nil {
			return "", err
		}
		if _, err := p.parseTypedValue(typ); err != nil {
			return "", err
		}
	case p.ident("section"), p.ident("partition"), p.ident("gc"):
		p.bump()
		if _, err := p.expectString(); err != nil {
			return "", err
		}
	case p.ident("align"), p.ident("alignstack"):
		p.bump()
		if _, err := p.integer(); err != nil {
			return "", err
		}
	case p.ident("comdat"):
		p.bump()
		if p.punct('(') {
			if err := p.skipDelim('(', ')'); err != nil {
				return "", err
			}
		}
	case p.ident("addrspace"):
		if _, err := p.parseAddrSpace(); err != nil {
			return "", err
		}
	case p.cur.kind == tokIdent:
		p.bump()
		if p.punct('(') {
			if err := p.skipDelim('(', ')'); err != nil {
				return "", err
			}
		}
	default:
		return "", p.syntax("function suffix")
	}
	return p.spanFrom(from), nil
}

func (p *parser) takeGlobalTrailer(g *Global) error {
	for p.acceptPunct(',') {
		if p.ident("align") {
			p.bump()
			n, err := p.integer()
			if err != nil {
				return err
			}
			g.Alignment = int(n)
			continue
		}
		if p.punct('!') {
			from := p.cur.start
			if _, err := p.takeAttach(); err != nil {
				return err
			}
			g.Suffix = append(g.Suffix, p.spanFrom(from))
			continue
		}
		if p.cur.kind == tokIdent {
			piece, err := p.takeSuffixPiece()
			if err != nil {
				return err
			}
			g.Suffix = append(g.Suffix, piece)
			continue
		}
		return p.unsupported("global trailer")
	}
	return nil
}

func (p *parser) takeGEPFlags(flags *[]string, inbounds *bool, nuw, nsw *bool) {
	for p.cur.kind == tokIdent {
		switch p.cur.text {
		case "inbounds":
			*inbounds = true
			p.bump()
		case "nuw":
			if nuw != nil {
				*nuw = true
			} else {
				*flags = append(*flags, "nuw")
			}
			p.bump()
		case "nsw":
			if nsw != nil {
				*nsw = true
			} else {
				*flags = append(*flags, "nsw")
			}
			p.bump()
		case "nusw":
			*flags = append(*flags, "nusw")
			p.bump()
		case "inrange":
			from := p.cur.start
			p.bump()
			if p.punct('(') {
				if err := p.skipDelim('(', ')'); err != nil {
					return
				}
				*flags = append(*flags, p.spanFrom(from))
				continue
			}
			*flags = append(*flags, "inrange")
		default:
			return
		}
	}
}

func (p *parser) takeSyncScope() (string, error) {
	if !p.ident("syncscope") {
		return "", nil
	}
	from := p.cur.start
	p.bump()
	if err := p.expectPunct('('); err != nil {
		return "", err
	}
	if p.cur.kind != tokString {
		return "", p.syntax("syncscope")
	}
	p.bump()
	if err := p.expectPunct(')'); err != nil {
		return "", err
	}
	return p.spanFrom(from), nil
}

func (p *parser) parseOrdering() (string, error) {
	if p.cur.kind != tokIdent {
		return "", p.syntax("ordering")
	}
	switch p.cur.text {
	case "unordered", "monotonic", "acquire", "release", "acq_rel", "seq_cst":
		text := p.cur.text
		p.bump()
		return text, nil
	default:
		return "", p.syntax("ordering")
	}
}

func isAtomicOp(name string) bool {
	switch name {
	case "xchg", "add", "sub", "and", "nand", "or", "xor", "max", "min", "umax", "umin",
		"fadd", "fsub", "fmax", "fmin", "uinc_wrap", "udec_wrap", "usub_cond", "usub_sat":
		return true
	default:
		return false
	}
}

func isOpcode(name string) bool {
	switch name {
	case "alloca", "load", "store", "getelementptr", "icmp", "fcmp", "phi",
		"extractvalue", "insertvalue", "call", "br", "ret", "unreachable",
		"tail", "musttail", "notail", "fneg", "freeze", "select",
		"extractelement", "insertelement", "shufflevector", "atomicrmw",
		"cmpxchg", "fence", "switch", "invoke", "resume", "landingpad",
		"callbr", "indirectbr", "va_arg", "catchswitch", "catchret",
		"cleanupret", "catchpad", "cleanuppad":
		return true
	}
	if _, err := ParseCast(name); err == nil {
		return true
	}
	if _, err := ParseOp(name); err == nil {
		return true
	}
	return false
}

func (p *parser) atInstBoundary() bool {
	if p.atLabel() || p.punct('}') || p.cur.kind == tokEOF || p.cur.kind == tokLocal {
		return true
	}
	return false
}

func (p *parser) takeCallSuffix(in *Inst) error {
	for !p.atInstBoundary() && !p.punct(',') {
		if p.ident("to") && p.nxt.kind == tokIdent && p.nxt.text == "label" {
			break
		}
		switch {
		case p.punct('#'):
			from := p.cur.start
			p.bump()
			if p.cur.kind != tokNumber {
				return p.syntax("attribute group")
			}
			p.bump()
			in.Groups = append(in.Groups, p.spanFrom(from))
		case p.punct('['):
			from := p.cur.start
			if err := p.skipDelim('[', ']'); err != nil {
				return err
			}
			in.Bundles = append(in.Bundles, p.spanFrom(from))
		case p.cur.kind == tokIdent && !isOpcode(p.cur.text):
			from := p.cur.start
			p.bump()
			if p.punct('(') {
				if err := p.skipDelim('(', ')'); err != nil {
					return err
				}
			}
			in.Groups = append(in.Groups, p.spanFrom(from))
		default:
			return nil
		}
	}
	return nil
}

func (p *parser) parseFNeg(in *Inst) error {
	in.Kind = InstFNeg
	in.Flags = p.takeFastMath()
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

func (p *parser) parseFreeze(in *Inst) error {
	in.Kind = InstFreeze
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

func (p *parser) parseSelect(in *Inst) error {
	in.Kind = InstSelect
	in.Flags = p.takeFastMath()
	condType, err := p.parseType()
	if err != nil {
		return err
	}
	in.CondType = condType
	cond, err := p.parseTypedValue(condType)
	if err != nil {
		return err
	}
	in.Condition = cond
	if err := p.expectPunct(','); err != nil {
		return err
	}
	arm, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = arm
	left, err := p.parseTypedValue(arm)
	if err != nil {
		return err
	}
	in.Left = left
	if err := p.expectPunct(','); err != nil {
		return err
	}
	// Both arms carry a type. The model keeps the first; legal IR uses one type.
	if _, err := p.parseType(); err != nil {
		return err
	}
	right, err := p.parseTypedValue(arm)
	if err != nil {
		return err
	}
	in.Right = right
	return p.skipTrailer(in, false)
}

func (p *parser) parseExtractElt(in *Inst) error {
	in.Kind = InstExtractElt
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
	idxType, err := p.parseType()
	if err != nil {
		return err
	}
	idx, err := p.parseTypedValue(idxType)
	if err != nil {
		return err
	}
	in.Index = &Operand{Type: idxType, Value: idx}
	return p.skipTrailer(in, false)
}

func (p *parser) parseInsertElt(in *Inst) error {
	in.Kind = InstInsertElt
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
	elemType, err := p.parseType()
	if err != nil {
		return err
	}
	elem, err := p.parseTypedValue(elemType)
	if err != nil {
		return err
	}
	in.Inserted = &Operand{Type: elemType, Value: elem}
	if err := p.expectPunct(','); err != nil {
		return err
	}
	idxType, err := p.parseType()
	if err != nil {
		return err
	}
	idx, err := p.parseTypedValue(idxType)
	if err != nil {
		return err
	}
	in.Index = &Operand{Type: idxType, Value: idx}
	return p.skipTrailer(in, false)
}

func (p *parser) parseShuffle(in *Inst) error {
	in.Kind = InstShuffle
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	left, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = left
	if err := p.expectPunct(','); err != nil {
		return err
	}
	otherType, err := p.parseType()
	if err != nil {
		return err
	}
	other, err := p.parseTypedValue(otherType)
	if err != nil {
		return err
	}
	in.Other = &Operand{Type: otherType, Value: other}
	if err := p.expectPunct(','); err != nil {
		return err
	}
	maskType, err := p.parseType()
	if err != nil {
		return err
	}
	mask, err := p.parseTypedValue(maskType)
	if err != nil {
		return err
	}
	in.Index = &Operand{Type: maskType, Value: mask}
	return p.skipTrailer(in, false)
}

func (p *parser) parseAtomicRMW(in *Inst) error {
	in.Kind = InstAtomicRMW
	if p.acceptIdent("volatile") {
		in.IsVolatile = true
	}
	scope, err := p.takeSyncScope()
	if err != nil {
		return err
	}
	in.SyncScope = scope
	if p.cur.kind != tokIdent || !isAtomicOp(p.cur.text) {
		return p.unsupported(p.cur.text)
	}
	in.AtomicOp = p.cur.text
	p.bump()
	ptrType, err := p.parseType()
	if err != nil {
		return err
	}
	in.Pointer = ptrType
	ptr, err := p.parseTypedValue(ptrType)
	if err != nil {
		return err
	}
	in.Left = ptr
	if err := p.expectPunct(','); err != nil {
		return err
	}
	valType, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = valType
	val, err := p.parseTypedValue(valType)
	if err != nil {
		return err
	}
	in.Right = val
	ord, err := p.parseOrdering()
	if err != nil {
		return err
	}
	in.Ordering = ord
	return p.skipTrailer(in, true)
}

func (p *parser) parseCmpXchg(in *Inst) error {
	in.Kind = InstCmpXchg
	if p.acceptIdent("weak") {
		in.Weak = true
	}
	if p.acceptIdent("volatile") {
		in.IsVolatile = true
	}
	if p.ident("weak") {
		in.Weak = true
		p.bump()
	}
	scope, err := p.takeSyncScope()
	if err != nil {
		return err
	}
	in.SyncScope = scope
	ptrType, err := p.parseType()
	if err != nil {
		return err
	}
	in.Pointer = ptrType
	ptr, err := p.parseTypedValue(ptrType)
	if err != nil {
		return err
	}
	in.Left = ptr
	if err := p.expectPunct(','); err != nil {
		return err
	}
	expType, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = expType
	exp, err := p.parseTypedValue(expType)
	if err != nil {
		return err
	}
	in.Right = exp
	if err := p.expectPunct(','); err != nil {
		return err
	}
	newType, err := p.parseType()
	if err != nil {
		return err
	}
	newVal, err := p.parseTypedValue(newType)
	if err != nil {
		return err
	}
	in.Inserted = &Operand{Type: newType, Value: newVal}
	ok, err := p.parseOrdering()
	if err != nil {
		return err
	}
	in.Ordering = ok
	fail, err := p.parseOrdering()
	if err != nil {
		return err
	}
	in.FailureOrd = fail
	return p.skipTrailer(in, true)
}

func (p *parser) parseFence(in *Inst) error {
	in.Kind = InstFence
	scope, err := p.takeSyncScope()
	if err != nil {
		return err
	}
	in.SyncScope = scope
	ord, err := p.parseOrdering()
	if err != nil {
		return err
	}
	in.Ordering = ord
	return p.skipTrailer(in, false)
}

func (p *parser) parseSwitch(in *Inst) error {
	in.Kind = InstSwitch
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	val, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = val
	if err := p.expectPunct(','); err != nil {
		return err
	}
	if err := p.expectIdent("label"); err != nil {
		return err
	}
	if p.cur.kind != tokLocal {
		return p.syntax("label")
	}
	in.Target = p.cur.text
	p.bump()
	if err := p.expectPunct('['); err != nil {
		return err
	}
	for !p.acceptPunct(']') {
		if p.cur.kind == tokEOF {
			return p.syntax("switch")
		}
		armType, err := p.parseType()
		if err != nil {
			return err
		}
		armVal, err := p.parseTypedValue(armType)
		if err != nil {
			return err
		}
		if err := p.expectPunct(','); err != nil {
			return err
		}
		if err := p.expectIdent("label"); err != nil {
			return err
		}
		if p.cur.kind != tokLocal {
			return p.syntax("label")
		}
		in.Arms = append(in.Arms, Arm{Type: armType, Value: armVal, Label: p.cur.text})
		p.bump()
	}
	return p.skipTrailer(in, false)
}

func (p *parser) parseInvoke(in *Inst) error {
	if err := p.parseCall(in); err != nil {
		return err
	}
	in.Kind = InstInvoke
	return p.parseUnwind(in, true)
}

func (p *parser) parseCallBr(in *Inst) error {
	if err := p.parseCall(in); err != nil {
		return err
	}
	in.Kind = InstCallBr
	if err := p.expectIdent("to"); err != nil {
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
	if err := p.expectPunct('['); err != nil {
		return err
	}
	for !p.acceptPunct(']') {
		if err := p.expectIdent("label"); err != nil {
			return err
		}
		if p.cur.kind != tokLocal {
			return p.syntax("label")
		}
		in.Labels = append(in.Labels, p.cur.text)
		p.bump()
	}
	return p.skipTrailer(in, false)
}

func (p *parser) parseUnwind(in *Inst, normal bool) error {
	if normal {
		if err := p.expectIdent("to"); err != nil {
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
	}
	if err := p.expectIdent("unwind"); err != nil {
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

func (p *parser) parseResume(in *Inst) error {
	in.Kind = InstResume
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

func (p *parser) parseLanding(in *Inst) error {
	in.Kind = InstLanding
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Type = typ
	for p.cur.kind == tokIdent {
		switch p.cur.text {
		case "cleanup":
			in.Clauses = append(in.Clauses, Clause{Kind: "cleanup"})
			p.bump()
		case "catch", "filter":
			kind := p.cur.text
			p.bump()
			clauseType, err := p.parseType()
			if err != nil {
				return err
			}
			clause, err := p.parseTypedValue(clauseType)
			if err != nil {
				return err
			}
			in.Clauses = append(in.Clauses, Clause{Kind: kind, Type: clauseType, Value: clause})
		default:
			return p.skipTrailer(in, false)
		}
	}
	return p.skipTrailer(in, false)
}

func (p *parser) parseIndirectBr(in *Inst) error {
	in.Kind = InstIndirectBr
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
	if err := p.expectPunct('['); err != nil {
		return err
	}
	for !p.acceptPunct(']') {
		if err := p.expectIdent("label"); err != nil {
			return err
		}
		if p.cur.kind != tokLocal {
			return p.syntax("label")
		}
		in.Labels = append(in.Labels, p.cur.text)
		p.bump()
		p.acceptPunct(',')
	}
	return p.skipTrailer(in, false)
}

func (p *parser) parseVAArg(in *Inst) error {
	in.Kind = InstVAArg
	typ, err := p.parseType()
	if err != nil {
		return err
	}
	in.Pointer = typ
	value, err := p.parseTypedValue(typ)
	if err != nil {
		return err
	}
	in.Left = value
	if err := p.expectPunct(','); err != nil {
		return err
	}
	dest, err := p.parseType()
	if err != nil {
		return err
	}
	in.Dest = dest
	in.Type = dest
	return p.skipTrailer(in, false)
}

func (p *parser) parseConstExpr() (*Value, error) {
	from := p.cur.start
	depth := 0
	saw := false
	steps := 0
	for {
		if p.cur.kind == tokEOF {
			return nil, p.syntax("constant expression")
		}
		if !saw {
			if steps > 6 {
				return nil, p.syntax("constant expression")
			}
			if steps > 0 && p.cur.kind != tokIdent && !p.punct('(') {
				return nil, p.syntax("constant expression")
			}
		}
		steps++
		if p.punct('(') {
			depth++
			saw = true
			p.bump()
			continue
		}
		if p.punct(')') {
			if !saw {
				return nil, p.syntax("constant expression")
			}
			depth--
			p.bump()
			if depth == 0 {
				break
			}
			continue
		}
		p.bump()
	}
	return &Value{Kind: ValExpr, Body: p.spanFrom(from)}, nil
}

func (p *parser) constExprStart() bool {
	if p.cur.kind != tokIdent || isAttr(p.cur.text) {
		return false
	}
	if p.nxt.kind == tokPunct && p.nxt.text == "(" {
		return true
	}
	switch p.cur.text {
	case "icmp", "fcmp", "select", "freeze", "fneg", "extractvalue", "insertvalue",
		"extractelement", "insertelement", "shufflevector", "blockaddress":
		return true
	}
	if _, err := ParseCast(p.cur.text); err == nil {
		return true
	}
	if _, err := ParseOp(p.cur.text); err == nil && p.nxt.kind == tokIdent {
		return true
	}
	return false
}

func (p *parser) parseMetaValue() (*Value, error) {
	from := p.cur.start
	if err := p.consumeMDValue(); err != nil {
		return nil, err
	}
	return &Value{Kind: ValMeta, Body: p.spanFrom(from)}, nil
}

func (p *parser) takeAtomicTail(in *Inst) error {
	if !in.Atomic {
		return nil
	}
	scope, err := p.takeSyncScope()
	if err != nil {
		return err
	}
	in.SyncScope = scope
	ord, err := p.parseOrdering()
	if err != nil {
		return err
	}
	in.Ordering = ord
	return nil
}
