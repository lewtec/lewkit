package c

// Kind is the shape of a C type.
//
//lewkit:enum
type Kind uint8

const (
	KindVoid Kind = iota + 1
	KindBool
	KindInteger
	KindFloat
	KindPointer
	KindArray
	KindFunction
	KindStruct
	KindUnion
	KindEnum
)

// DeclKind is one top-level item in a translation unit.
//
//lewkit:enum
type DeclKind uint8

const (
	DeclInclude DeclKind = iota + 1
	DeclType
	DeclConst
	DeclFunc
	DeclObject
)

// Storage is a storage-class specifier.
//
//lewkit:enum
type Storage uint8

const (
	StorageStatic Storage = iota + 1
	StorageExtern
)

// StmtKind is one statement in a function body.
//
//lewkit:enum
type StmtKind uint8

const (
	StmtBlock StmtKind = iota + 1
	StmtDecl
	StmtExpr
	StmtReturn
	StmtIf
	StmtWhile
	StmtFor
	StmtBreak
	StmtContinue
)

// ExprKind is one expression.
//
//lewkit:enum
type ExprKind uint8

const (
	ExprIdent ExprKind = iota + 1
	ExprInt
	ExprFloat
	ExprString
	ExprChar
	ExprCall
	ExprUnary
	ExprBinary
	ExprAssign
	ExprField
	ExprIndex
	ExprCast
	ExprSizeof
)

// Op is a unary or binary operator.
//
//lewkit:enum
type Op uint8

const (
	OpAdd Op = iota + 1
	OpSub
	OpMul
	OpDiv
	OpMod
	OpEq
	OpNe
	OpLt
	OpGt
	OpLe
	OpGe
	OpAnd
	OpOr
	OpBitAnd
	OpBitOr
	OpBitXor
	OpShl
	OpShr
	OpNot
	OpBitNot
	OpNeg
	OpPos
	OpAddr
	OpDeref
)
