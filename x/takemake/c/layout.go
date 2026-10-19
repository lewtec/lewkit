package c

// LP64 System V. long and pointer are 8 bytes. long double is 16.
const (
	ptrSize        = 8
	intSize        = 4
	longSize       = 8
	longDoubleSize = 16
)

func (f *File) layout() {
	if f == nil {
		return
	}
	for _, d := range f.Decls {
		switch d.Kind {
		case DeclType:
			if d.Type != nil {
				layout(d.Type.Type)
			}
		case DeclFunc:
			layoutFunc(d.Func)
		case DeclObject:
			layoutObject(d.Object)
		}
	}
}

func layoutFunc(fn *Func) {
	if fn == nil {
		return
	}
	layout(fn.Result)
	for _, param := range fn.Params {
		if param != nil {
			layout(param.Type)
		}
	}
	layoutStmt(fn.Body)
}

func layoutObject(obj *Object) {
	if obj == nil {
		return
	}
	layout(obj.Type)
	layoutExpr(obj.Init)
}

func layoutStmt(s *Stmt) {
	if s == nil {
		return
	}
	layoutExpr(s.Expr)
	layoutExpr(s.Post)
	layoutStmt(s.Body)
	layoutStmt(s.Else)
	layoutStmt(s.Init)
	for _, child := range s.Stmts {
		layoutStmt(child)
	}
	for _, obj := range s.Objects {
		layoutObject(obj)
	}
}

func layoutExpr(e *Expr) {
	if e == nil {
		return
	}
	layout(e.Type)
	layoutExpr(e.Left)
	layoutExpr(e.Right)
	for _, arg := range e.Args {
		layoutExpr(arg)
	}
}

func layout(t *Type) {
	if t == nil || t.laying {
		return
	}
	t.laying = true
	defer func() { t.laying = false }()
	if t.Of != nil {
		layout(t.Of)
		t.Size = t.Of.Size
		t.Align = t.Of.Align
		if t.Align < 1 {
			t.Align = 1
		}
		return
	}
	switch t.Kind {
	case KindVoid:
		t.Size = 0
		t.Align = 1
	case KindBool:
		t.Size = 1
		t.Align = 1
	case KindInteger, KindFloat:
		if t.Align < 1 {
			t.Align = t.Size
		}
		if t.Align < 1 {
			t.Align = 1
		}
	case KindPointer:
		if t.Elem != nil && !t.Elem.laying {
			layout(t.Elem)
		}
		t.Size = ptrSize
		t.Align = ptrSize
	case KindArray:
		layout(t.Elem)
		elemSize, elemAlign := 0, 1
		if t.Elem != nil {
			elemSize = t.Elem.Size
			if t.Elem.Align > 1 {
				elemAlign = t.Elem.Align
			}
		}
		t.Align = elemAlign
		if t.Len < 0 {
			t.Size = 0
			return
		}
		t.Size = t.Len * elemSize
	case KindFunction:
		layout(t.Elem)
		for _, param := range t.Params {
			if param != nil {
				layout(param.Type)
			}
		}
		t.Size = 0
		t.Align = 1
	case KindStruct:
		layoutStruct(t)
	case KindUnion:
		layoutUnion(t)
	case KindEnum:
		layoutEnum(t)
	}
}

func layoutStruct(t *Type) {
	offset := 0
	align := 1
	for _, field := range t.Fields {
		if field == nil {
			continue
		}
		layout(field.Type)
		fieldAlign := 1
		fieldSize := 0
		if field.Type != nil {
			if field.Type.Align > 1 {
				fieldAlign = field.Type.Align
			}
			fieldSize = field.Type.Size
		}
		offset = alignUp(offset, fieldAlign)
		field.Offset = offset
		offset += fieldSize
		if fieldAlign > align {
			align = fieldAlign
		}
	}
	t.Align = align
	t.Size = alignUp(offset, align)
}

func layoutUnion(t *Type) {
	size := 0
	align := 1
	for _, field := range t.Fields {
		if field == nil {
			continue
		}
		layout(field.Type)
		field.Offset = 0
		if field.Type == nil {
			continue
		}
		if field.Type.Size > size {
			size = field.Type.Size
		}
		if field.Type.Align > align {
			align = field.Type.Align
		}
	}
	t.Align = align
	t.Size = alignUp(size, align)
}

func layoutEnum(t *Type) {
	size := intSize
	for _, item := range t.Enums {
		if item == nil {
			continue
		}
		if item.Value > int32Max || item.Value < int32Min {
			size = 8
		}
	}
	t.Size = size
	t.Align = size
}

func alignUp(n, align int) int {
	if align <= 1 {
		return n
	}
	return (n + align - 1) / align * align
}

const (
	int32Max = 2147483647
	int32Min = -2147483648
)
