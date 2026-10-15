package llvm

import (
	"fmt"
	"io"
	"strconv"
)

// sink records the first write error and ignores later writes.
type sink struct {
	w   io.Writer
	err error
}

func (s *sink) raw(text string) {
	if s.err != nil || text == "" {
		return
	}
	_, s.err = io.WriteString(s.w, text)
}

func (s *sink) nl() { s.raw("\n") }

func (f *File) validate() error {
	if f == nil {
		return ErrNilFile
	}
	for _, def := range f.Types {
		if def == nil || def.Name == "" {
			return ErrModule
		}
		if def.IsOpaque {
			continue
		}
		if err := validType(def.Body); err != nil {
			return err
		}
	}
	for _, g := range f.Globals {
		if err := validGlobal(g); err != nil {
			return err
		}
	}
	for _, fn := range f.Funcs {
		if err := validFunc(fn); err != nil {
			return err
		}
	}
	return nil
}

func validGlobal(g *Global) error {
	if g == nil || g.Name == "" {
		return ErrModule
	}
	if err := knownLinkage(g.Linkage); err != nil {
		return err
	}
	if err := knownUnnamed(g.Unnamed); err != nil {
		return err
	}
	if err := validType(g.ValueType); err != nil {
		return err
	}
	if g.Initializer != nil {
		return validValue(g.Initializer)
	}
	return nil
}

func validFunc(fn *Func) error {
	if fn == nil || fn.Name == "" {
		return ErrModule
	}
	if err := knownLinkage(fn.Linkage); err != nil {
		return err
	}
	if err := knownUnnamed(fn.Unnamed); err != nil {
		return err
	}
	if err := validType(fn.Result); err != nil {
		return err
	}
	for _, param := range fn.Parameters {
		if param == nil {
			return ErrModule
		}
		if err := validType(param.Type); err != nil {
			return err
		}
	}
	if !fn.Defined {
		return nil
	}
	for _, block := range fn.Blocks {
		if err := validBlock(block); err != nil {
			return err
		}
	}
	return nil
}

func validBlock(block *Block) error {
	if block == nil || block.Name == "" {
		return ErrModule
	}
	for _, in := range block.Instructions {
		if err := validInst(in); err != nil {
			return err
		}
	}
	return nil
}

func validInst(in *Inst) error {
	if in == nil {
		return ErrModule
	}
	if !in.Kind.IsValid() {
		return fmt.Errorf("%w: %q", ErrInstKind, in.Kind)
	}
	if in.Tail != "" && !in.Tail.IsValid() {
		return fmt.Errorf("%w: %q", ErrTail, in.Tail)
	}
	switch in.Kind {
	case InstAlloca:
		if err := validType(in.Type); err != nil {
			return err
		}
		if in.Count != nil {
			return validOperand(*in.Count)
		}
	case InstLoad:
		if err := validType(in.Type); err != nil {
			return err
		}
		if err := validType(in.Pointer); err != nil {
			return err
		}
		return validValue(in.Left)
	case InstStore:
		if err := validType(in.Type); err != nil {
			return err
		}
		if err := validValue(in.Left); err != nil {
			return err
		}
		if err := validType(in.Pointer); err != nil {
			return err
		}
		return validValue(in.Right)
	case InstGEP:
		if err := validType(in.Source); err != nil {
			return err
		}
		if err := validType(in.Pointer); err != nil {
			return err
		}
		if err := validValue(in.Left); err != nil {
			return err
		}
		return validOperands(in.Indices)
	case InstBin:
		if !in.Op.IsValid() {
			return fmt.Errorf("%w: %q", ErrOp, in.Op)
		}
		if err := validType(in.Type); err != nil {
			return err
		}
		if err := validValue(in.Left); err != nil {
			return err
		}
		return validValue(in.Right)
	case InstICmp:
		if !in.IntegerPred.IsValid() {
			return fmt.Errorf("%w: %q", ErrIPred, in.IntegerPred)
		}
		return validCmp(in)
	case InstFCmp:
		if !in.FloatPred.IsValid() {
			return fmt.Errorf("%w: %q", ErrFPred, in.FloatPred)
		}
		return validCmp(in)
	case InstCast:
		if !in.Cast.IsValid() {
			return fmt.Errorf("%w: %q", ErrCast, in.Cast)
		}
		if err := validType(in.Type); err != nil {
			return err
		}
		if err := validValue(in.Left); err != nil {
			return err
		}
		return validType(in.Dest)
	case InstPhi:
		if err := validType(in.Type); err != nil {
			return err
		}
		if len(in.Incoming) == 0 {
			return ErrModule
		}
		for _, edge := range in.Incoming {
			if edge.Block == "" {
				return ErrModule
			}
			if err := validValue(edge.Value); err != nil {
				return err
			}
		}
	case InstExtract:
		if err := validType(in.Type); err != nil {
			return err
		}
		if err := validValue(in.Left); err != nil {
			return err
		}
		if len(in.Path) == 0 {
			return ErrModule
		}
	case InstInsert:
		if err := validType(in.Type); err != nil {
			return err
		}
		if err := validValue(in.Left); err != nil {
			return err
		}
		if in.Inserted == nil {
			return ErrModule
		}
		if err := validOperand(*in.Inserted); err != nil {
			return err
		}
		if len(in.Path) == 0 {
			return ErrModule
		}
	case InstCall:
		if err := validType(in.Type); err != nil {
			return err
		}
		if in.Signature != nil {
			if err := validType(in.Signature); err != nil {
				return err
			}
		}
		if err := validValue(in.Callee); err != nil {
			return err
		}
		return validOperands(in.Args)
	case InstBr:
		if in.Condition == nil {
			if in.Target == "" {
				return ErrModule
			}
			return nil
		}
		if in.TrueLabel == "" || in.FalseLabel == "" {
			return ErrModule
		}
		return validValue(in.Condition)
	case InstRet:
		if err := validType(in.Type); err != nil {
			return err
		}
		if in.Type.Kind == KindVoid {
			return nil
		}
		return validValue(in.Left)
	case InstUnreachable:
		return nil
	}
	return nil
}

func validCmp(in *Inst) error {
	if err := validType(in.Type); err != nil {
		return err
	}
	if err := validValue(in.Left); err != nil {
		return err
	}
	return validValue(in.Right)
}

func validOperands(ops []Operand) error {
	for _, op := range ops {
		if err := validOperand(op); err != nil {
			return err
		}
	}
	return nil
}

func validOperand(op Operand) error {
	if err := validType(op.Type); err != nil {
		return err
	}
	return validValue(op.Value)
}

func validValue(v *Value) error {
	if v == nil {
		return ErrModule
	}
	if !v.Kind.IsValid() {
		return fmt.Errorf("%w: %q", ErrValueKind, v.Kind)
	}
	switch v.Kind {
	case ValLocal, ValGlobal:
		if v.Name == "" {
			return ErrModule
		}
	case ValInt, ValFloat:
		if v.Text == "" {
			return ErrModule
		}
		if v.Type != nil {
			return validType(v.Type)
		}
	case ValNull, ValZero, ValUndef, ValPoison:
		if v.Type != nil {
			return validType(v.Type)
		}
	case ValArray, ValStruct, ValPacked, ValVector:
		return validOperands(v.Elems)
	case ValGEP:
		if err := validType(v.SourceType); err != nil {
			return err
		}
		if err := validType(v.PointerType); err != nil {
			return err
		}
		if err := validValue(v.Base); err != nil {
			return err
		}
		return validOperands(v.Indices)
	}
	return nil
}

func validType(t *Type) error {
	if t == nil {
		return ErrModule
	}
	if !t.Kind.IsValid() {
		return fmt.Errorf("%w: %q", ErrKind, t.Kind)
	}
	switch t.Kind {
	case KindInt:
		if t.Bits <= 0 {
			return ErrModule
		}
	case KindFloat:
		if !t.Float.IsValid() {
			return fmt.Errorf("%w: %q", ErrFloat, t.Float)
		}
	case KindPtr:
		if t.Elem != nil {
			return validType(t.Elem)
		}
	case KindArray, KindVector:
		if t.Len < 0 {
			return ErrModule
		}
		return validType(t.Elem)
	case KindStruct:
		for _, field := range t.Fields {
			if err := validType(field); err != nil {
				return err
			}
		}
	case KindFunc:
		if err := validType(t.Ret); err != nil {
			return err
		}
		for _, field := range t.Fields {
			if err := validType(field); err != nil {
				return err
			}
		}
	case KindNamed:
		if t.Name == "" {
			return ErrModule
		}
	}
	return nil
}

func knownLinkage(link Linkage) error {
	if link != "" && !link.IsValid() {
		return fmt.Errorf("%w: %q", ErrLinkage, link)
	}
	return nil
}

func knownUnnamed(addr Unnamed) error {
	if addr != "" && !addr.IsValid() {
		return fmt.Errorf("%w: %q", ErrUnnamed, addr)
	}
	return nil
}

func (f *File) write(s *sink) {
	wrote := false
	if f.SourceFilename != "" {
		s.raw("source_filename = ")
		s.quoted(f.SourceFilename)
		s.nl()
		wrote = true
	}
	if f.DataLayout != "" {
		s.raw("target datalayout = ")
		s.quoted(f.DataLayout)
		s.nl()
		wrote = true
	}
	if f.TargetTriple != "" {
		s.raw("target triple = ")
		s.quoted(f.TargetTriple)
		s.nl()
		wrote = true
	}
	if len(f.Types) > 0 {
		if wrote {
			s.nl()
		}
		for _, def := range f.Types {
			s.typeDef(def)
		}
		wrote = true
	}
	if len(f.Globals) > 0 {
		if wrote {
			s.nl()
		}
		for _, g := range f.Globals {
			s.global(g)
		}
		wrote = true
	}
	for i, fn := range f.Funcs {
		if wrote || i > 0 {
			s.nl()
		}
		s.function(fn)
		wrote = true
	}
}

func (s *sink) typeDef(def *TypeDef) {
	s.sigil('%', def.Name)
	s.raw(" = type ")
	if def.IsOpaque {
		s.raw("opaque")
	} else {
		s.typ(def.Body)
	}
	s.nl()
}

func (s *sink) global(g *Global) {
	s.sigil('@', g.Name)
	s.raw(" =")
	if g.Linkage != "" {
		s.raw(" ")
		s.raw(string(g.Linkage))
	}
	if g.DSOLocal {
		s.raw(" dso_local")
	}
	if g.Unnamed != "" {
		s.raw(" ")
		s.raw(string(g.Unnamed))
	}
	if g.IsConstant {
		s.raw(" constant ")
	} else {
		s.raw(" global ")
	}
	s.typ(g.ValueType)
	if g.Initializer != nil {
		s.raw(" ")
		s.value(g.Initializer)
	}
	s.align(g.Alignment)
	s.nl()
}

func (s *sink) function(fn *Func) {
	if fn.Defined {
		s.raw("define")
	} else {
		s.raw("declare")
	}
	if fn.Linkage != "" {
		s.raw(" ")
		s.raw(string(fn.Linkage))
	}
	if fn.DSOLocal {
		s.raw(" dso_local")
	}
	s.raw(" ")
	s.typ(fn.Result)
	s.raw(" ")
	s.sigil('@', fn.Name)
	s.raw("(")
	for i, param := range fn.Parameters {
		if i > 0 {
			s.raw(", ")
		}
		s.typ(param.Type)
		if param.Name != "" {
			s.raw(" ")
			s.sigil('%', param.Name)
		}
	}
	if fn.Variadic {
		if len(fn.Parameters) > 0 {
			s.raw(", ")
		}
		s.raw("...")
	}
	s.raw(")")
	if fn.Unnamed != "" {
		s.raw(" ")
		s.raw(string(fn.Unnamed))
	}
	if !fn.Defined {
		s.nl()
		return
	}
	s.raw(" {\n")
	for _, block := range fn.Blocks {
		s.name(block.Name)
		s.raw(":\n")
		for _, in := range block.Instructions {
			s.raw("\t")
			s.inst(in)
			s.nl()
		}
	}
	s.raw("}\n")
}

func (s *sink) inst(in *Inst) {
	switch in.Kind {
	case InstAlloca:
		s.assign(in.Name)
		s.raw("alloca ")
		s.typ(in.Type)
		if in.Count != nil {
			s.raw(", ")
			s.operand(*in.Count)
		}
		s.align(in.Alignment)
	case InstLoad:
		s.assign(in.Name)
		s.raw("load ")
		if in.IsVolatile {
			s.raw("volatile ")
		}
		s.typ(in.Type)
		s.raw(", ")
		s.typ(in.Pointer)
		s.raw(" ")
		s.value(in.Left)
		s.align(in.Alignment)
	case InstStore:
		s.raw("store ")
		if in.IsVolatile {
			s.raw("volatile ")
		}
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
		s.raw(", ")
		s.typ(in.Pointer)
		s.raw(" ")
		s.value(in.Right)
		s.align(in.Alignment)
	case InstGEP:
		s.assign(in.Name)
		s.raw("getelementptr ")
		if in.InBounds {
			s.raw("inbounds ")
		}
		s.typ(in.Source)
		s.raw(", ")
		s.typ(in.Pointer)
		s.raw(" ")
		s.value(in.Left)
		for _, index := range in.Indices {
			s.raw(", ")
			s.operand(index)
		}
		s.align(in.Alignment)
	case InstBin:
		s.assign(in.Name)
		s.raw(string(in.Op))
		// nuw, then nsw, then exact. The source order does not matter.
		if in.NoUnsignedWrap {
			s.raw(" nuw")
		}
		if in.NoSignedWrap {
			s.raw(" nsw")
		}
		if in.ExactDiv {
			s.raw(" exact")
		}
		s.raw(" ")
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
		s.raw(", ")
		s.value(in.Right)
	case InstICmp:
		s.assign(in.Name)
		s.raw("icmp ")
		s.raw(string(in.IntegerPred))
		s.raw(" ")
		s.typedValues(in)
	case InstFCmp:
		s.assign(in.Name)
		s.raw("fcmp ")
		s.raw(string(in.FloatPred))
		s.raw(" ")
		s.typedValues(in)
	case InstCast:
		s.assign(in.Name)
		s.raw(string(in.Cast))
		s.raw(" ")
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
		s.raw(" to ")
		s.typ(in.Dest)
	case InstPhi:
		s.assign(in.Name)
		s.raw("phi ")
		s.typ(in.Type)
		for i, edge := range in.Incoming {
			if i > 0 {
				s.raw(",")
			}
			s.raw(" [ ")
			s.value(edge.Value)
			s.raw(", ")
			s.sigil('%', edge.Block)
			s.raw(" ]")
		}
	case InstExtract:
		s.assign(in.Name)
		s.raw("extractvalue ")
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
		s.path(in.Path)
	case InstInsert:
		s.assign(in.Name)
		s.raw("insertvalue ")
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
		s.raw(", ")
		s.operand(*in.Inserted)
		s.path(in.Path)
	case InstCall:
		s.assign(in.Name)
		if in.Tail != "" {
			s.raw(string(in.Tail))
			s.raw(" ")
		}
		s.raw("call ")
		if in.Signature != nil {
			s.typ(in.Signature)
		} else {
			s.typ(in.Type)
		}
		s.raw(" ")
		s.value(in.Callee)
		s.raw("(")
		for i, arg := range in.Args {
			if i > 0 {
				s.raw(", ")
			}
			s.operand(arg)
		}
		s.raw(")")
	case InstBr:
		s.raw("br ")
		if in.Condition == nil {
			s.raw("label ")
			s.sigil('%', in.Target)
			return
		}
		s.raw("i1 ")
		s.value(in.Condition)
		s.raw(", label ")
		s.sigil('%', in.TrueLabel)
		s.raw(", label ")
		s.sigil('%', in.FalseLabel)
	case InstRet:
		s.raw("ret ")
		if in.Type != nil && in.Type.Kind == KindVoid {
			s.raw("void")
			return
		}
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
	case InstUnreachable:
		s.raw("unreachable")
	}
}

func (s *sink) assign(name string) {
	if name == "" {
		return
	}
	s.sigil('%', name)
	s.raw(" = ")
}

func (s *sink) typedValues(in *Inst) {
	s.typ(in.Type)
	s.raw(" ")
	s.value(in.Left)
	s.raw(", ")
	s.value(in.Right)
}

func (s *sink) path(path []int) {
	for _, index := range path {
		s.raw(", ")
		s.raw(strconv.Itoa(index))
	}
}

func (s *sink) align(n int) {
	if n == 0 {
		return
	}
	s.raw(", align ")
	s.raw(strconv.Itoa(n))
}

func (s *sink) operand(op Operand) {
	s.typ(op.Type)
	s.raw(" ")
	s.value(op.Value)
}

func (s *sink) value(v *Value) {
	if v == nil {
		return
	}
	switch v.Kind {
	case ValLocal:
		s.sigil('%', v.Name)
	case ValGlobal:
		s.sigil('@', v.Name)
	case ValInt, ValFloat:
		s.raw(v.Text)
	case ValNull:
		s.raw("null")
	case ValZero:
		s.raw("zeroinitializer")
	case ValUndef:
		s.raw("undef")
	case ValPoison:
		s.raw("poison")
	case ValTrue:
		s.raw("true")
	case ValFalse:
		s.raw("false")
	case ValString:
		s.raw("c")
		s.quoted(v.Text)
	case ValStruct:
		s.aggregate("{", "}", v.Elems)
	case ValPacked:
		s.raw("<")
		s.aggregate("{", "}", v.Elems)
		s.raw(">")
	case ValArray:
		s.aggregate("[", "]", v.Elems)
	case ValVector:
		s.aggregate("<", ">", v.Elems)
	case ValGEP:
		s.raw("getelementptr ")
		if v.InBounds {
			s.raw("inbounds ")
		}
		s.raw("(")
		s.typ(v.SourceType)
		s.raw(", ")
		s.typ(v.PointerType)
		s.raw(" ")
		s.value(v.Base)
		for _, index := range v.Indices {
			s.raw(", ")
			s.operand(index)
		}
		s.raw(")")
	}
}

func (s *sink) aggregate(open, close string, elems []Operand) {
	s.raw(open)
	if len(elems) > 0 {
		s.raw(" ")
	}
	for i, elem := range elems {
		if i > 0 {
			s.raw(", ")
		}
		s.operand(elem)
	}
	if len(elems) > 0 {
		s.raw(" ")
	}
	s.raw(close)
}

// typ prints one type. An opaque pointer is `ptr`. A typed pointer keeps
// the element and a trailing star, so the two forms stay distinct.
func (s *sink) typ(t *Type) {
	if t == nil {
		return
	}
	switch t.Kind {
	case KindVoid:
		s.raw("void")
	case KindInt:
		s.raw("i")
		s.raw(strconv.Itoa(t.Bits))
	case KindFloat:
		s.raw(string(t.Float))
	case KindPtr:
		if t.Elem == nil {
			s.raw("ptr")
			s.addrSpace(t.AddrSpace)
			return
		}
		s.typ(t.Elem)
		s.addrSpace(t.AddrSpace)
		s.raw("*")
	case KindArray:
		s.raw("[")
		s.raw(strconv.FormatInt(t.Len, 10))
		s.raw(" x ")
		s.typ(t.Elem)
		s.raw("]")
	case KindVector:
		s.raw("<")
		s.raw(strconv.FormatInt(t.Len, 10))
		s.raw(" x ")
		s.typ(t.Elem)
		s.raw(">")
	case KindStruct:
		if t.Packed {
			s.raw("<")
			s.fields("{", "}", t.Fields)
			s.raw(">")
			return
		}
		s.fields("{", "}", t.Fields)
	case KindFunc:
		s.typ(t.Ret)
		s.raw(" (")
		for i, param := range t.Fields {
			if i > 0 {
				s.raw(", ")
			}
			s.typ(param)
		}
		if t.Variadic {
			if len(t.Fields) > 0 {
				s.raw(", ")
			}
			s.raw("...")
		}
		s.raw(")")
	case KindNamed:
		s.sigil('%', t.Name)
	}
}

func (s *sink) fields(open, close string, fields []*Type) {
	s.raw(open)
	if len(fields) > 0 {
		s.raw(" ")
	}
	for i, field := range fields {
		if i > 0 {
			s.raw(", ")
		}
		s.typ(field)
	}
	if len(fields) > 0 {
		s.raw(" ")
	}
	s.raw(close)
}

func (s *sink) addrSpace(space int) {
	if space == 0 {
		return
	}
	s.raw(" addrspace(")
	s.raw(strconv.Itoa(space))
	s.raw(")")
}

func (s *sink) sigil(mark byte, name string) {
	s.raw(string(mark))
	s.name(name)
}

func (s *sink) name(text string) {
	if bareName(text) {
		s.raw(text)
		return
	}
	s.quoted(text)
}

func (s *sink) quoted(text string) {
	s.raw(`"`)
	s.escape(text)
	s.raw(`"`)
}

func (s *sink) escape(text string) {
	for i := 0; i < len(text); i++ {
		c := text[i]
		switch {
		case c == '\\' || c == '"':
			s.raw(`\`)
			s.raw(string(c))
		case c < 0x20 || c > 0x7e:
			s.raw(fmt.Sprintf("\\%02X", c))
		default:
			s.raw(string(c))
		}
	}
}

func bareName(name string) bool {
	if name == "" {
		return false
	}
	digits := true
	for i := 0; i < len(name); i++ {
		if !isDigit(name[i]) {
			digits = false
			break
		}
	}
	if digits {
		return true
	}
	if !isSigilStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isSigilCont(name[i]) {
			return false
		}
	}
	return true
}
