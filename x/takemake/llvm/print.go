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
	for _, comdat := range f.Comdats {
		if comdat == nil || comdat.Name == "" || !comdat.Kind.IsValid() {
			return ErrModule
		}
	}
	for _, group := range f.AttrGroups {
		if group == nil || group.ID == "" || group.Body == "" {
			return ErrModule
		}
	}
	for _, meta := range f.Metadata {
		if meta == nil || meta.Name == "" || meta.RHS == "" {
			return ErrModule
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
	if g.Aliasee != nil {
		if err := validOperand(*g.Aliasee); err != nil {
			return err
		}
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
	if block == nil {
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
	case ValExpr, ValMeta:
		if v.Body == "" {
			return ErrModule
		}
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
	case KindNamed, KindPrim:
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
	if len(f.Asms) > 0 {
		if wrote {
			s.nl()
		}
		for _, asm := range f.Asms {
			s.raw("module asm ")
			s.quoted(asm)
			s.nl()
		}
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
	if len(f.Comdats) > 0 {
		if wrote {
			s.nl()
		}
		for _, comdat := range f.Comdats {
			s.comdat(comdat)
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
	if len(f.AttrGroups) > 0 {
		if wrote {
			s.nl()
		}
		for _, group := range f.AttrGroups {
			s.raw("attributes #")
			s.raw(group.ID)
			s.raw(" = ")
			s.raw(group.Body)
			s.nl()
		}
		wrote = true
	}
	if len(f.Metadata) > 0 {
		if wrote {
			s.nl()
		}
		for _, meta := range f.Metadata {
			s.raw("!")
			s.name(meta.Name)
			s.raw(" = ")
			s.raw(meta.RHS)
			s.nl()
		}
	}
}

func (s *sink) comdat(comdat *Comdat) {
	if bareName(comdat.Name) {
		s.raw("$")
		s.raw(comdat.Name)
	} else {
		s.raw(`$"`)
		s.escape(comdat.Name)
		s.raw(`"`)
	}
	s.raw(" = comdat ")
	s.raw(string(comdat.Kind))
	s.nl()
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
	s.attrs(g.Prefix)
	if g.Unnamed != "" {
		s.raw(" ")
		s.raw(string(g.Unnamed))
	}
	switch {
	case g.Alias:
		s.raw(" alias ")
	case g.IFunc:
		s.raw(" ifunc ")
	case g.IsConstant:
		s.raw(" constant ")
	default:
		s.raw(" global ")
	}
	s.typ(g.ValueType)
	if g.Alias || g.IFunc {
		s.raw(", ")
		if g.Aliasee != nil {
			s.operand(*g.Aliasee)
		}
	} else if g.Initializer != nil {
		s.afterType(g.Initializer)
	}
	s.align(g.Alignment)
	for _, piece := range g.Suffix {
		s.raw(", ")
		s.raw(piece)
	}
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
	s.attrs(fn.Prefix)
	if fn.Conv != "" {
		s.raw(" ")
		s.raw(fn.Conv)
	}
	s.attrs(fn.ResultAttrs)
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
		s.attrs(param.Attrs)
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
	s.attrs(fn.Suffix)
	for _, att := range fn.Attach {
		s.raw(" !")
		s.name(att.Name)
		s.raw(" ")
		s.raw(att.Body)
	}
	if !fn.Defined {
		s.nl()
		return
	}
	s.raw(" {\n")
	for _, block := range fn.Blocks {
		if block.Name != "" {
			s.name(block.Name)
			s.raw(":\n")
		}
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
		s.lead(in.Flags)
		s.typ(in.Type)
		if in.Count != nil {
			s.raw(", ")
			s.operand(*in.Count)
		}
		s.finish(in, true)
	case InstLoad:
		s.assign(in.Name)
		s.raw("load ")
		if in.Atomic {
			s.raw("atomic ")
		}
		if in.IsVolatile {
			s.raw("volatile ")
		}
		s.typ(in.Type)
		s.raw(", ")
		s.typ(in.Pointer)
		s.afterType(in.Left)
		s.atomicOrder(in)
		s.finish(in, true)
	case InstStore:
		s.raw("store ")
		if in.Atomic {
			s.raw("atomic ")
		}
		if in.IsVolatile {
			s.raw("volatile ")
		}
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", ")
		s.typ(in.Pointer)
		s.afterType(in.Right)
		s.atomicOrder(in)
		s.finish(in, true)
	case InstGEP:
		s.assign(in.Name)
		s.gepHead(in.InBounds, in.NoUnsignedWrap, in.NoSignedWrap, in.Flags)
		s.typ(in.Source)
		s.raw(", ")
		s.typ(in.Pointer)
		s.afterType(in.Left)
		for _, index := range in.Indices {
			s.raw(", ")
			s.operand(index)
		}
		s.finish(in, true)
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
		s.attrs(in.Flags)
		s.raw(" ")
		s.typ(in.Type)
		s.raw(" ")
		s.value(in.Left)
		s.raw(", ")
		s.value(in.Right)
		s.finish(in, false)
	case InstICmp:
		s.assign(in.Name)
		s.raw("icmp ")
		s.lead(in.Flags)
		s.raw(string(in.IntegerPred))
		s.raw(" ")
		s.typedValues(in)
		s.finish(in, false)
	case InstFCmp:
		s.assign(in.Name)
		s.raw("fcmp ")
		s.lead(in.Flags)
		s.raw(string(in.FloatPred))
		s.raw(" ")
		s.typedValues(in)
		s.finish(in, false)
	case InstCast:
		s.assign(in.Name)
		s.raw(string(in.Cast))
		if in.NoUnsignedWrap {
			s.raw(" nuw")
		}
		if in.NoSignedWrap {
			s.raw(" nsw")
		}
		s.attrs(in.Flags)
		s.raw(" ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(" to ")
		s.typ(in.Dest)
		s.finish(in, false)
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
		s.finish(in, false)
	case InstExtract:
		s.assign(in.Name)
		s.raw("extractvalue ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.path(in.Path)
		s.finish(in, false)
	case InstInsert:
		s.assign(in.Name)
		s.raw("insertvalue ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", ")
		s.operand(*in.Inserted)
		s.path(in.Path)
		s.finish(in, false)
	case InstCall, InstInvoke, InstCallBr:
		s.callLike(in)
		s.finish(in, false)
	case InstBr:
		s.raw("br ")
		if in.Condition == nil {
			s.raw("label ")
			s.sigil('%', in.Target)
		} else {
			s.raw("i1")
			s.afterType(in.Condition)
			s.raw(", label ")
			s.sigil('%', in.TrueLabel)
			s.raw(", label ")
			s.sigil('%', in.FalseLabel)
		}
		s.finish(in, false)
	case InstRet:
		s.raw("ret ")
		if in.Type != nil && in.Type.Kind == KindVoid {
			s.raw("void")
		} else {
			s.typ(in.Type)
			s.afterType(in.Left)
		}
		s.finish(in, false)
	case InstUnreachable:
		s.raw("unreachable")
		s.finish(in, false)
	case InstFNeg:
		s.assign(in.Name)
		s.raw("fneg ")
		s.lead(in.Flags)
		s.typ(in.Type)
		s.afterType(in.Left)
		s.finish(in, false)
	case InstFreeze:
		s.assign(in.Name)
		s.raw("freeze ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.finish(in, false)
	case InstSelect:
		s.assign(in.Name)
		s.raw("select ")
		s.lead(in.Flags)
		s.typ(in.CondType)
		s.afterType(in.Condition)
		s.raw(", ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", ")
		s.typ(in.Type)
		s.afterType(in.Right)
		s.finish(in, false)
	case InstExtractElt:
		s.assign(in.Name)
		s.raw("extractelement ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", ")
		s.operand(*in.Index)
		s.finish(in, false)
	case InstInsertElt:
		s.assign(in.Name)
		s.raw("insertelement ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", ")
		s.operand(*in.Inserted)
		s.raw(", ")
		s.operand(*in.Index)
		s.finish(in, false)
	case InstShuffle:
		s.assign(in.Name)
		s.raw("shufflevector ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", ")
		s.operand(*in.Other)
		s.raw(", ")
		s.operand(*in.Index)
		s.finish(in, false)
	case InstAtomicRMW:
		s.assign(in.Name)
		s.raw("atomicrmw ")
		if in.IsVolatile {
			s.raw("volatile ")
		}
		s.leadOne(in.SyncScope)
		s.raw(in.AtomicOp)
		s.raw(" ")
		s.typ(in.Pointer)
		s.afterType(in.Left)
		s.raw(", ")
		s.typ(in.Type)
		s.afterType(in.Right)
		s.raw(" ")
		s.raw(in.Ordering)
		s.finish(in, true)
	case InstCmpXchg:
		s.assign(in.Name)
		s.raw("cmpxchg ")
		if in.Weak {
			s.raw("weak ")
		}
		if in.IsVolatile {
			s.raw("volatile ")
		}
		s.leadOne(in.SyncScope)
		s.typ(in.Pointer)
		s.afterType(in.Left)
		s.raw(", ")
		s.typ(in.Type)
		s.afterType(in.Right)
		s.raw(", ")
		s.operand(*in.Inserted)
		s.raw(" ")
		s.raw(in.Ordering)
		s.raw(" ")
		s.raw(in.FailureOrd)
		s.finish(in, true)
	case InstFence:
		s.assign(in.Name)
		s.raw("fence ")
		s.leadOne(in.SyncScope)
		s.raw(in.Ordering)
		s.finish(in, false)
	case InstSwitch:
		s.assign(in.Name)
		s.raw("switch ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", label ")
		s.sigil('%', in.Target)
		s.raw(" [")
		for _, arm := range in.Arms {
			s.raw(" ")
			s.typ(arm.Type)
			s.afterType(arm.Value)
			s.raw(", label ")
			s.sigil('%', arm.Label)
		}
		s.raw(" ]")
		s.finish(in, false)
	case InstResume:
		s.assign(in.Name)
		s.raw("resume ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.finish(in, false)
	case InstLanding:
		s.assign(in.Name)
		s.raw("landingpad ")
		s.typ(in.Type)
		for _, clause := range in.Clauses {
			s.raw(" ")
			s.raw(clause.Kind)
			if clause.Value != nil {
				s.raw(" ")
				s.typ(clause.Type)
				s.afterType(clause.Value)
			}
		}
		s.finish(in, false)
	case InstIndirectBr:
		s.assign(in.Name)
		s.raw("indirectbr ")
		s.typ(in.Type)
		s.afterType(in.Left)
		s.raw(", [")
		for i, label := range in.Labels {
			if i > 0 {
				s.raw(",")
			}
			s.raw(" label ")
			s.sigil('%', label)
		}
		s.raw(" ]")
		s.finish(in, false)
	case InstVAArg:
		s.assign(in.Name)
		s.raw("va_arg ")
		s.typ(in.Pointer)
		s.afterType(in.Left)
		s.raw(", ")
		s.typ(in.Dest)
		s.finish(in, false)
	}
}

func (s *sink) assign(name string) {
	if name == "" {
		return
	}
	s.sigil('%', name)
	s.raw(" = ")
}

// attrs prints each clause with a leading space. A nil slice prints nothing.
func (s *sink) attrs(words []string) {
	for _, word := range words {
		s.raw(" ")
		s.raw(word)
	}
}

// lead prints each clause followed by a space.
func (s *sink) lead(words []string) {
	for _, word := range words {
		s.raw(word)
		s.raw(" ")
	}
}

func (s *sink) leadOne(word string) {
	if word == "" {
		return
	}
	s.raw(word)
	s.raw(" ")
}

// afterType prints value attributes, then the value. The type is already written.
func (s *sink) afterType(v *Value) {
	if v != nil {
		s.attrs(v.Attrs)
	}
	s.raw(" ")
	s.value(v)
}

func (s *sink) atomicOrder(in *Inst) {
	if !in.Atomic {
		return
	}
	if in.SyncScope != "" {
		s.raw(" ")
		s.raw(in.SyncScope)
	}
	if in.Ordering != "" {
		s.raw(" ")
		s.raw(in.Ordering)
	}
}

func (s *sink) attaches(list []Attach) {
	for _, att := range list {
		s.raw(", !")
		s.name(att.Name)
		s.raw(" ")
		s.raw(att.Body)
	}
}

func (s *sink) finish(in *Inst, align bool) {
	if align {
		s.align(in.Alignment)
	}
	s.attaches(in.Attach)
}

func (s *sink) gepHead(inbounds, nuw, nsw bool, flags []string) {
	s.raw("getelementptr ")
	if inbounds {
		s.raw("inbounds ")
	}
	if nuw {
		s.raw("nuw ")
	}
	if nsw {
		s.raw("nsw ")
	}
	s.lead(flags)
}

func (s *sink) callLike(in *Inst) {
	s.assign(in.Name)
	if in.Tail != "" {
		s.raw(string(in.Tail))
		s.raw(" ")
	}
	switch in.Kind {
	case InstInvoke:
		s.raw("invoke ")
	case InstCallBr:
		s.raw("callbr ")
	default:
		s.raw("call ")
	}
	s.lead(in.Flags)
	s.leadOne(in.Conv)
	s.lead(in.ResultAttrs)
	if in.Signature != nil {
		s.typ(in.Signature)
	} else {
		s.typ(in.Type)
	}
	if in.Callee != nil {
		s.attrs(in.Callee.Attrs)
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
	s.attrs(in.Groups)
	s.attrs(in.Bundles)
	switch in.Kind {
	case InstInvoke:
		s.raw(" to label ")
		s.sigil('%', in.TrueLabel)
		s.raw(" unwind label ")
		s.sigil('%', in.FalseLabel)
	case InstCallBr:
		s.raw(" to label ")
		s.sigil('%', in.TrueLabel)
		s.raw(" [")
		for i, label := range in.Labels {
			if i > 0 {
				s.raw(",")
			}
			s.raw(" label ")
			s.sigil('%', label)
		}
		s.raw(" ]")
	}
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
	if op.Value != nil {
		s.attrs(op.Value.Attrs)
	}
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
		s.gepHead(v.InBounds, false, false, v.Flags)
		s.raw("(")
		s.typ(v.SourceType)
		s.raw(", ")
		s.typ(v.PointerType)
		s.afterType(v.Base)
		for _, index := range v.Indices {
			s.raw(", ")
			s.operand(index)
		}
		s.raw(")")
	case ValExpr, ValMeta:
		s.raw(v.Body)
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
		if t.Scalable {
			s.raw("vscale x ")
		}
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
	case KindMeta:
		s.raw("metadata")
	case KindToken:
		s.raw("token")
	case KindLabel:
		s.raw("label")
	case KindPrim:
		s.raw(t.Name)
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
