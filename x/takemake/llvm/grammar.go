package llvm

import (
	"fmt"
	"strings"
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

func language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		lang, langErr = define()
	})
	return lang, langErr
}

type gram struct {
	syntax xparse.Syntax[Term, Rule]
	t      map[Term]*xparse.Piece[Term, Rule]
	seen   map[string]bool
}

func (g *gram) tok(kind Term) *xparse.Piece[Term, Rule] {
	if p, ok := g.t[kind]; ok {
		return p
	}
	p := g.syntax.Term(kind, nil)
	g.t[kind] = p
	return p
}

func (g *gram) ref(name Rule) *xparse.Piece[Term, Rule] {
	return g.syntax.Ref(name)
}

func (g *gram) lit(text string) *xparse.Piece[Term, Rule] {
	g.seen[text] = true
	return g.syntax.Lit(text)
}

func (g *gram) lits(words []string) *xparse.Piece[Term, Rule] {
	got := map[string]bool{}
	alts := make([]*xparse.Piece[Term, Rule], 0, len(words))
	for _, word := range words {
		if got[word] {
			continue
		}
		got[word] = true
		alts = append(alts, g.lit(word))
	}
	return g.syntax.Alt(alts...)
}

func (g *gram) comma() *xparse.Piece[Term, Rule] { return g.lit(",") }

func define() (*xparse.Model[Term, Rule], error) {
	g := &gram{t: map[Term]*xparse.Piece[Term, Rule]{}, seen: map[string]bool{}}
	syntax := g.syntax
	var parts []*xparse.Piece[Term, Rule]
	equation := func(name Rule, body *xparse.Piece[Term, Rule]) {
		parts = append(parts, syntax.Eq(name, body))
	}

	identifier := g.tok(TermIdentifier)
	number := g.tok(TermNumber)
	stringToken := g.tok(TermString)
	cString := g.tok(TermCString)
	localReference := g.tok(TermLocalReference)
	globalReference := g.tok(TermGlobalReference)
	localDefinition := g.tok(TermLocalDefinition)
	globalDefinition := g.tok(TermGlobalDefinition)
	integerType := g.tok(TermIntegerType)
	comdatName := g.tok(TermComdatName)
	labelName := g.tok(TermLabelName)
	exclamation := g.tok(TermExclamation)
	metadataExclamation := g.tok(TermMetadataExclamation)
	attributeGroupReference := g.tok(TermAttributeGroupReference)
	braced := g.tok(TermBraced)
	ellipsis := g.tok(TermEllipsis)
	leftParenthesis := g.lit("(")
	rightParenthesis := g.lit(")")
	leftBrace := g.lit("{")
	rightBrace := g.lit("}")
	leftBracket := g.lit("[")
	rightBracket := g.lit("]")
	less := g.lit("<")
	greater := g.lit(">")
	star := g.lit("*")
	hash := g.lit("#")
	equalToken := g.lit("=")

	linkage := []string{
		"external", "private", "internal", "available_externally", "linkonce",
		"weak", "common", "appending", "extern_weak", "linkonce_odr", "weak_odr",
	}
	callingConventionWords := []string{
		"ccc", "fastcc", "coldcc", "ghccc", "webkit_jscc", "anyregcc",
		"preserve_mostcc", "preserve_allcc", "preserve_nonecc", "cxx_fast_tlscc",
		"tailcc", "swiftcc", "swifttailcc", "cfguard_checkcc", "nakedcc",
		"intel_ocl_bicc", "x86_stdcallcc", "x86_fastcallcc", "x86_thiscallcc",
		"x86_vectorcallcc", "x86_regcallcc", "arm_apcscc", "arm_aapcscc",
		"arm_aapcs_vfpcc", "aarch64_vector_pcs", "aarch64_sve_vector_pcs",
		"msp430_intrcc", "avr_intrcc", "avr_signalcc", "ptx_kernel", "ptx_device",
		"spir_func", "spir_kernel", "x86_64_sysvcc", "win64cc", "amdgpu_vs",
		"amdgpu_ls", "amdgpu_hs", "amdgpu_es", "amdgpu_gs", "amdgpu_ps",
		"amdgpu_cs", "amdgpu_kernel", "amdgpu_gfx", "hhvmcc", "hhvm_ccc",
	}
	fastMathWords := []string{"nnan", "ninf", "nsz", "arcp", "contract", "afn", "reassoc", "fast"}
	order := []string{"unordered", "monotonic", "acquire", "release", "acq_rel", "seq_cst"}
	binaryWords := []string{
		"add", "fadd", "sub", "fsub", "mul", "fmul", "udiv", "sdiv", "fdiv",
		"urem", "srem", "frem", "shl", "lshr", "ashr", "and", "or", "xor",
	}
	cast := []string{
		"trunc", "zext", "sext", "fptrunc", "fpext", "fptoui", "fptosi", "uitofp",
		"sitofp", "ptrtoint", "inttoptr", "bitcast", "addrspacecast",
	}
	integerPredicateWords := []string{"eq", "ne", "ugt", "uge", "ult", "ule", "sgt", "sge", "slt", "sle"}
	floatPredicateWords := []string{
		"false", "oeq", "ogt", "oge", "olt", "ole", "one", "ord",
		"ueq", "ugt", "uge", "ult", "ule", "une", "uno", "true",
	}
	atomicWords := []string{
		"xchg", "add", "sub", "and", "nand", "or", "xor", "max", "min", "umax", "umin",
		"fadd", "fsub", "fmax", "fmin", "uinc_wrap", "udec_wrap", "usub_cond", "usub_sat",
	}
	primitiveWords := []string{
		"void", "half", "float", "double", "metadata", "token", "label",
		"bfloat", "x86_fp80", "fp128", "ppc_fp128", "x86_mmx", "x86_amx",
	}
	getElementPointerFlagWords := []string{"inbounds", "nuw", "nsw", "nusw", "inrange"}
	binFlag := append(append([]string{}, "nuw", "nsw", "exact", "disjoint"), fastMathWords...)
	castFlag := append(append([]string{}, "nuw", "nsw", "nneg"), fastMathWords...)

	link := g.lits(linkage)
	callingConventions := g.lits(callingConventionWords)
	fast := g.lits(fastMathWords)
	ordering := g.lits(order)
	binaryOperators := g.lits(binaryWords)
	casts := g.lits(cast)
	integerPredicates := g.lits(integerPredicateWords)
	floatPredicates := g.lits(floatPredicateWords)
	atomicOperators := g.lits(atomicWords)
	primitiveTypes := g.lits(primitiveWords)
	getElementPointerFlags := g.lits(getElementPointerFlagWords)
	binaryFlags := g.lits(binFlag)
	castFlags := g.lits(castFlag)

	equation(RuleCallingConventionNumber, syntax.Seq(g.lit("cc"), number))
	equation(RuleSync, syntax.Seq(g.lit("syncscope"), leftParenthesis, stringToken, rightParenthesis))
	equation(RuleAddressSpace, syntax.Seq(g.lit("addrspace"), leftParenthesis, number, rightParenthesis))
	equation(RuleWord, syntax.Alt(
		identifier,
		attributeGroupReference,
		syntax.Seq(g.lit("align"), number),
		syntax.Seq(g.lit("alignstack"), number),
		g.lit("nneg"),
		g.lit("inalloca"),
	))
	equation(RuleStar, syntax.Alt(star, syntax.Seq(g.ref(RuleAddressSpace), star)))
	equation(RuleMetadataNode, syntax.Seq(
		syntax.Opt(g.lit("distinct")),
		exclamation,
		syntax.Alt(number, stringToken, identifier, attributeGroupReference, braced),
	))
	equation(RuleAttach, syntax.Seq(exclamation, syntax.Alt(identifier, number), g.ref(RuleMetadataNode)))

	typeList := syntax.Opt(syntax.Alt(
		ellipsis,
		syntax.Seq(g.ref(RuleType), syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(g.ref(RuleType), ellipsis)))),
	))
	equation(RuleFunctionTail, syntax.Seq(leftParenthesis, typeList, rightParenthesis, syntax.Repeat(g.ref(RuleStar))))

	array := syntax.Seq(leftBracket, number, identifier, g.ref(RuleType), rightBracket)
	fieldList := tyFields(g)
	angled := syntax.Seq(less, syntax.Alt(
		syntax.Seq(leftBrace, fieldList, rightBrace, greater),
		syntax.Seq(identifier, identifier, number, identifier, g.ref(RuleType), greater),
		syntax.Seq(number, identifier, g.ref(RuleType), greater),
	))
	structType := syntax.Seq(leftBrace, fieldList, rightBrace)
	simple := syntax.Alt(integerType, primitiveTypes, localReference, array, angled)
	ptrForm := func(bareStar *xparse.Piece[Term, Rule]) *xparse.Piece[Term, Rule] {
		return syntax.Seq(g.lit("ptr"), syntax.Opt(g.ref(RuleAddressSpace)), syntax.Repeat(bareStar), syntax.Opt(g.ref(RuleFunctionTail)))
	}
	other := func(core *xparse.Piece[Term, Rule]) *xparse.Piece[Term, Rule] {
		return syntax.Seq(core, syntax.Repeat(g.ref(RuleStar)), syntax.Opt(g.ref(RuleFunctionTail)))
	}
	equation(RuleType, syntax.Alt(ptrForm(star), other(syntax.Alt(structType, simple))))
	equation(RuleTypeWithoutStruct, syntax.Alt(ptrForm(star), other(simple)))

	equation(RuleValue, syntax.Alt(
		localReference, globalReference, number, cString,
		g.lit("true"), g.lit("false"), g.lit("null"), g.lit("undef"),
		g.lit("poison"), g.lit("zeroinitializer"),
		g.ref(RuleGetElementPointer), g.ref(RuleConstantExpression), g.ref(RuleAggregate), g.ref(RuleMetadataNode),
	))
	equation(RuleTypedValue, syntax.Seq(syntax.Repeat(g.ref(RuleWord)), g.ref(RuleValue)))
	equation(RuleOperand, syntax.Seq(g.ref(RuleType), g.ref(RuleTypedValue)))
	equation(RuleOperandWithoutStruct, syntax.Seq(g.ref(RuleTypeWithoutStruct), g.ref(RuleTypedValue)))

	elemList := syntax.Opt(syntax.Seq(g.ref(RuleOperand), syntax.Repeat(syntax.Seq(g.comma(), g.ref(RuleOperand)))))
	vecList := syntax.Opt(syntax.Seq(g.ref(RuleOperandWithoutStruct), syntax.Repeat(syntax.Seq(g.comma(), g.ref(RuleOperand)))))
	equation(RuleAggregate, syntax.Alt(
		syntax.Seq(leftBrace, elemList, rightBrace),
		syntax.Seq(leftBracket, elemList, rightBracket),
		syntax.Seq(less, syntax.Alt(
			syntax.Seq(leftBrace, elemList, rightBrace, greater),
			syntax.Seq(vecList, greater),
		)),
	))
	equation(RuleGetElementPointer, syntax.Seq(
		g.lit("getelementptr"),
		syntax.Repeat(getElementPointerFlags),
		leftParenthesis,
		g.ref(RuleType),
		g.comma(),
		g.ref(RuleOperand),
		syntax.Repeat(syntax.Seq(g.comma(), g.ref(RuleOperand))),
		rightParenthesis,
	))
	equation(RuleConstantExpression, syntax.Seq(
		casts,
		syntax.Repeat(castFlags),
		leftParenthesis,
		g.ref(RuleType),
		g.ref(RuleTypedValue),
		g.lit("to"),
		g.ref(RuleType),
		rightParenthesis,
	))

	equation(RulePiece, syntax.Alt(
		syntax.Seq(g.lit("personality"), g.ref(RuleType), g.ref(RuleTypedValue)),
		syntax.Seq(g.lit("prefix"), g.ref(RuleType), g.ref(RuleTypedValue)),
		syntax.Seq(g.lit("prologue"), g.ref(RuleType), g.ref(RuleTypedValue)),
		syntax.Seq(g.lit("section"), stringToken),
		syntax.Seq(g.lit("partition"), stringToken),
		syntax.Seq(g.lit("gc"), stringToken),
		syntax.Seq(g.lit("align"), number),
		syntax.Seq(g.lit("alignstack"), number),
		g.lit("comdat"),
		g.ref(RuleAddressSpace),
		syntax.Seq(hash, number),
		identifier,
		attributeGroupReference,
	))

	attachTrail := syntax.Repeat(syntax.Seq(g.comma(), g.ref(RuleAttach)))
	alignTrail := syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(
		syntax.Seq(g.lit("align"), number),
		g.ref(RuleAttach),
	)))
	mixedTrail := syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(
		g.ref(RuleOperand),
		syntax.Seq(g.lit("align"), number),
		g.ref(RuleAttach),
	)))
	numberTrail := syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(number, g.ref(RuleAttach))))
	phiArm := syntax.Seq(leftBracket, g.ref(RuleValue), g.comma(), localReference, rightBracket)
	labelPair := syntax.Seq(g.lit("label"), localReference, g.comma(), g.lit("label"), localReference)

	callSuf := syntax.Alt(syntax.Seq(hash, number), identifier, attributeGroupReference)
	equation(RuleCallBody, syntax.Seq(
		syntax.Repeat(fast),
		syntax.Opt(syntax.Alt(callingConventions, g.ref(RuleCallingConventionNumber))),
		syntax.Repeat(g.ref(RuleWord)),
		g.ref(RuleType),
		g.ref(RuleTypedValue),
		leftParenthesis,
		syntax.Opt(syntax.Seq(g.ref(RuleOperand), syntax.Repeat(syntax.Seq(g.comma(), g.ref(RuleOperand))))),
		rightParenthesis,
		syntax.Repeat(callSuf),
	))

	assign := syntax.Seq(localDefinition, equalToken)
	ops := [][]*xparse.Piece[Term, Rule]{
		{g.lit("alloca"), syntax.Opt(g.lit("inalloca")), g.ref(RuleType), mixedTrail},
		{g.lit("load"), syntax.Opt(g.lit("atomic")), syntax.Opt(g.lit("volatile")), g.ref(RuleType), g.comma(), g.ref(RuleOperand), syntax.Opt(g.ref(RuleSync)), syntax.Opt(ordering), alignTrail},
		{g.lit("store"), syntax.Opt(g.lit("atomic")), syntax.Opt(g.lit("volatile")), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), syntax.Opt(g.ref(RuleSync)), syntax.Opt(ordering), alignTrail},
		{g.lit("getelementptr"), syntax.Repeat(getElementPointerFlags), g.ref(RuleType), g.comma(), g.ref(RuleOperand), mixedTrail},
		{binaryOperators, syntax.Repeat(binaryFlags), g.ref(RuleType), g.ref(RuleValue), g.comma(), g.ref(RuleValue), attachTrail},
		{g.lit("icmp"), syntax.Repeat(identifier), integerPredicates, g.ref(RuleType), g.ref(RuleValue), g.comma(), g.ref(RuleValue), attachTrail},
		{g.lit("fcmp"), syntax.Repeat(fast), floatPredicates, g.ref(RuleType), g.ref(RuleValue), g.comma(), g.ref(RuleValue), attachTrail},
		{casts, syntax.Repeat(castFlags), g.ref(RuleType), g.ref(RuleTypedValue), g.lit("to"), g.ref(RuleType), attachTrail},
		{g.lit("phi"), g.ref(RuleType), phiArm, syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(phiArm, g.ref(RuleAttach))))},
		{g.lit("extractvalue"), g.ref(RuleOperand), numberTrail},
		{g.lit("insertvalue"), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), numberTrail},
		{syntax.Opt(syntax.Alt(g.lit("tail"), g.lit("musttail"), g.lit("notail"))), g.lit("call"), g.ref(RuleCallBody), attachTrail},
		{g.lit("invoke"), g.ref(RuleCallBody), g.lit("to"), g.lit("label"), localReference, g.lit("unwind"), g.lit("label"), localReference, attachTrail},
		{g.lit("callbr"), g.ref(RuleCallBody), g.lit("to"), g.lit("label"), localReference, leftBracket, syntax.Repeat(syntax.Seq(g.lit("label"), localReference, syntax.Opt(g.comma()))), rightBracket, attachTrail},
		{g.lit("br"), g.ref(RuleType), g.ref(RuleTypedValue), syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(labelPair, g.ref(RuleAttach))))},
		{g.lit("ret"), g.ref(RuleType), syntax.Opt(g.ref(RuleTypedValue)), attachTrail},
		{g.lit("unreachable"), attachTrail},
		{g.lit("fneg"), syntax.Repeat(fast), g.ref(RuleType), g.ref(RuleTypedValue), attachTrail},
		{g.lit("freeze"), g.ref(RuleType), g.ref(RuleTypedValue), attachTrail},
		{g.lit("select"), syntax.Repeat(fast), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), g.comma(), g.ref(RuleType), g.ref(RuleTypedValue), attachTrail},
		{g.lit("extractelement"), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), attachTrail},
		{g.lit("insertelement"), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), attachTrail},
		{g.lit("shufflevector"), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), attachTrail},
		{g.lit("atomicrmw"), syntax.Opt(g.lit("volatile")), syntax.Opt(g.ref(RuleSync)), atomicOperators, g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), ordering, alignTrail},
		{g.lit("cmpxchg"), syntax.Repeat(syntax.Alt(g.lit("weak"), g.lit("volatile"))), syntax.Opt(g.ref(RuleSync)), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), g.comma(), g.ref(RuleOperand), ordering, ordering, alignTrail},
		{g.lit("fence"), syntax.Opt(g.ref(RuleSync)), ordering, attachTrail},
		{g.lit("switch"), g.ref(RuleOperand), g.comma(), g.lit("label"), localReference, leftBracket, syntax.Repeat(syntax.Seq(g.ref(RuleOperand), g.comma(), g.lit("label"), localReference)), rightBracket, attachTrail},
		{g.lit("resume"), g.ref(RuleOperand), attachTrail},
		{g.lit("landingpad"), g.ref(RuleType), syntax.Repeat(syntax.Alt(
			g.lit("cleanup"),
			syntax.Seq(g.lit("catch"), g.ref(RuleOperand)),
			syntax.Seq(g.lit("filter"), g.ref(RuleOperand)),
		)), attachTrail},
		{g.lit("indirectbr"), g.ref(RuleOperand), g.comma(), leftBracket, syntax.Repeat(syntax.Seq(g.lit("label"), localReference, syntax.Opt(g.comma()))), rightBracket, attachTrail},
		{g.lit("va_arg"), g.ref(RuleOperand), g.comma(), g.ref(RuleType), attachTrail},
	}
	alts := make([]*xparse.Piece[Term, Rule], 0, len(ops))
	for _, op := range ops {
		alts = append(alts, syntax.Seq(op...))
	}
	equation(RuleInstruction, syntax.Seq(syntax.Opt(assign), syntax.Alt(alts...)))

	unnamed := syntax.Alt(g.lit("unnamed_addr"), g.lit("local_unnamed_addr"))
	functionPrefix := syntax.Alt(link, g.lit("dso_local"), unnamed, callingConventions, g.ref(RuleCallingConventionNumber), g.ref(RuleWord))
	functionSuffix := syntax.Alt(unnamed, g.ref(RuleAttach), g.ref(RulePiece))
	paramList := syntax.Opt(syntax.Alt(
		ellipsis,
		syntax.Seq(g.ref(RuleParameter), syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(g.ref(RuleParameter), ellipsis)))),
	))
	equation(RuleParameter, syntax.Seq(g.ref(RuleType), syntax.Repeat(g.ref(RuleWord)), syntax.Opt(localReference)))
	equation(RuleParameters, syntax.Seq(leftParenthesis, paramList, rightParenthesis))
	equation(RuleHeader, syntax.Seq(syntax.Repeat(functionPrefix), g.ref(RuleType), globalReference, g.ref(RuleParameters), syntax.Repeat(functionSuffix)))
	equation(RuleBody, syntax.Seq(leftBrace, syntax.Repeat(syntax.Alt(labelName, g.ref(RuleInstruction))), rightBrace))
	equation(RuleFunction, syntax.Alt(
		syntax.Seq(g.lit("define"), g.ref(RuleHeader), g.ref(RuleBody)),
		syntax.Seq(g.lit("declare"), g.ref(RuleHeader)),
	))

	globalPrefix := syntax.Alt(link, g.lit("dso_local"), unnamed, g.ref(RuleWord))
	globalTrail := syntax.Repeat(syntax.Seq(g.comma(), syntax.Alt(g.ref(RulePiece), g.ref(RuleAttach))))
	equation(RuleGlobalVariable, syntax.Seq(globalDefinition, equalToken, syntax.Repeat(globalPrefix), syntax.Alt(
		syntax.Seq(syntax.Alt(g.lit("global"), g.lit("constant")), g.ref(RuleType), syntax.Opt(g.ref(RuleTypedValue)), globalTrail),
		syntax.Seq(syntax.Alt(g.lit("alias"), g.lit("ifunc")), g.ref(RuleType), g.comma(), g.ref(RuleOperand), globalTrail),
	)))

	equation(RuleSource, syntax.Seq(g.lit("source_filename"), equalToken, stringToken))
	equation(RuleLayout, syntax.Seq(g.lit("target"), syntax.Alt(g.lit("datalayout"), g.lit("triple")), equalToken, stringToken))
	equation(RuleModuleAssembly, syntax.Seq(g.lit("module"), g.lit("asm"), stringToken))
	equation(RuleTypedef, syntax.Seq(localDefinition, equalToken, g.lit("type"), syntax.Alt(g.lit("opaque"), g.ref(RuleType))))
	equation(RuleTopMetadata, syntax.Seq(metadataExclamation, syntax.Alt(identifier, number), equalToken, g.ref(RuleMetadataNode)))
	equation(RuleAttributeGroup, syntax.Seq(g.lit("attributes"), hash, number, equalToken, braced))
	equation(RuleComdatDefinition, syntax.Seq(comdatName, equalToken, g.lit("comdat"), identifier))
	equation(RuleFile, syntax.Repeat(syntax.Alt(
		g.ref(RuleSource),
		g.ref(RuleLayout),
		g.ref(RuleModuleAssembly),
		g.ref(RuleTypedef),
		g.ref(RuleGlobalVariable),
		g.ref(RuleFunction),
		g.ref(RuleTopMetadata),
		g.ref(RuleAttributeGroup),
		g.ref(RuleComdatDefinition),
	)))

	var missing []string
	for _, word := range keywordList {
		if !g.seen[word] {
			missing = append(missing, word)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: grammar missed keywords %s", ErrSyntax, strings.Join(missing, " "))
	}
	parts = append(parts, syntax.Scan(scanLLVM), syntax.Start(RuleFile))
	return syntax.Compile("llvm", syntax.Compose(parts...))
}

// tyFields is a comma-separated type list. The closing braced stays in the parent.
func tyFields(g *gram) *xparse.Piece[Term, Rule] {
	return g.syntax.Opt(g.syntax.Seq(g.ref(RuleType), g.syntax.Repeat(g.syntax.Seq(g.comma(), g.ref(RuleType)))))
}
