package golang

import (
	"fmt"
	"go/ast"
	"go/token"
)

// ExprKind is the shape of an expression.
//
//lewkit:enum
type ExprKind uint8

const (
	ExprIdent ExprKind = iota + 1
	ExprSelector
	ExprCall
	ExprLit
	ExprComposite
	ExprUnary
	ExprBinary
	ExprIndex
	ExprIndexList
	ExprSlice
	ExprParen
	ExprType
	ExprFunc
	ExprAssert
)

// LitKind is a literal's form. Bool and nil are literals here. The
// parser spells them as identifiers.
//
//lewkit:enum
type LitKind uint8

const (
	LitInt LitKind = iota + 1
	LitFloat
	LitImag
	LitRune
	LitString
	LitBool
	LitNil
)

// TypeKind is the shape of a type.
//
//lewkit:enum
type TypeKind uint8

const (
	TypeNamed TypeKind = iota + 1
	TypePointer
	TypeSlice
	TypeArray
	TypeMap
	TypeStruct
	TypeInterface
	TypeFunc
	TypeChan
	TypeInst
	TypeBinary
	TypeUnary
)

// ChanDir is a channel direction. Both is a bidirectional channel.
//
//lewkit:enum
type ChanDir uint8

const (
	ChanBoth ChanDir = iota + 1
	ChanSend
	ChanRecv
)

// StmtKind is the shape of a statement.
//
//lewkit:enum
type StmtKind uint8

const (
	StmtExpr StmtKind = iota + 1
	StmtReturn
	StmtAssign
	StmtIf
	StmtFor
	StmtRange
	StmtBlock
	StmtDecl
	StmtInc
	StmtDec
	StmtDefer
	StmtGo
	StmtBranch
	StmtSend
	StmtLabel
	StmtSwitch
	StmtTypeSwitch
	StmtSelect
	StmtCase
	StmtComm
)

// Branch is a break, continue, goto, or fallthrough.
//
//lewkit:enum
type Branch uint8

const (
	BranchBreak Branch = iota + 1
	BranchContinue
	BranchGoto
	BranchFallthrough
)

// Op is a unary or binary operator. The same operator is unary or
// binary depending on the expression kind.
//
//lewkit:enum
type Op uint8

const (
	OpAdd Op = iota + 1
	OpSub
	OpMul
	OpQuo
	OpRem
	OpAnd
	OpOr
	OpXor
	OpShl
	OpShr
	OpAndNot
	OpLand
	OpLor
	OpArrow
	OpEql
	OpNeq
	OpLss
	OpLeq
	OpGtr
	OpGeq
	OpNot
	OpTilde
)

func opFrom(tok token.Token) (Op, error) {
	switch tok {
	case token.ADD:
		return OpAdd, nil
	case token.SUB:
		return OpSub, nil
	case token.MUL:
		return OpMul, nil
	case token.QUO:
		return OpQuo, nil
	case token.REM:
		return OpRem, nil
	case token.AND:
		return OpAnd, nil
	case token.OR:
		return OpOr, nil
	case token.XOR:
		return OpXor, nil
	case token.SHL:
		return OpShl, nil
	case token.SHR:
		return OpShr, nil
	case token.AND_NOT:
		return OpAndNot, nil
	case token.LAND:
		return OpLand, nil
	case token.LOR:
		return OpLor, nil
	case token.ARROW:
		return OpArrow, nil
	case token.EQL:
		return OpEql, nil
	case token.NEQ:
		return OpNeq, nil
	case token.LSS:
		return OpLss, nil
	case token.LEQ:
		return OpLeq, nil
	case token.GTR:
		return OpGtr, nil
	case token.GEQ:
		return OpGeq, nil
	case token.NOT:
		return OpNot, nil
	case token.TILDE:
		return OpTilde, nil
	default:
		return 0, fmt.Errorf("%w: operator %s", errSyntax, tok)
	}
}

func (op Op) token() (token.Token, error) {
	switch op {
	case OpAdd:
		return token.ADD, nil
	case OpSub:
		return token.SUB, nil
	case OpMul:
		return token.MUL, nil
	case OpQuo:
		return token.QUO, nil
	case OpRem:
		return token.REM, nil
	case OpAnd:
		return token.AND, nil
	case OpOr:
		return token.OR, nil
	case OpXor:
		return token.XOR, nil
	case OpShl:
		return token.SHL, nil
	case OpShr:
		return token.SHR, nil
	case OpAndNot:
		return token.AND_NOT, nil
	case OpLand:
		return token.LAND, nil
	case OpLor:
		return token.LOR, nil
	case OpArrow:
		return token.ARROW, nil
	case OpEql:
		return token.EQL, nil
	case OpNeq:
		return token.NEQ, nil
	case OpLss:
		return token.LSS, nil
	case OpLeq:
		return token.LEQ, nil
	case OpGtr:
		return token.GTR, nil
	case OpGeq:
		return token.GEQ, nil
	case OpNot:
		return token.NOT, nil
	case OpTilde:
		return token.TILDE, nil
	default:
		return 0, fmt.Errorf("%w: operator", errSyntax)
	}
}

func assignToken(op Op) (token.Token, error) {
	if op == 0 {
		return token.ASSIGN, nil
	}
	switch op {
	case OpAdd:
		return token.ADD_ASSIGN, nil
	case OpSub:
		return token.SUB_ASSIGN, nil
	case OpMul:
		return token.MUL_ASSIGN, nil
	case OpQuo:
		return token.QUO_ASSIGN, nil
	case OpRem:
		return token.REM_ASSIGN, nil
	case OpAnd:
		return token.AND_ASSIGN, nil
	case OpOr:
		return token.OR_ASSIGN, nil
	case OpXor:
		return token.XOR_ASSIGN, nil
	case OpShl:
		return token.SHL_ASSIGN, nil
	case OpShr:
		return token.SHR_ASSIGN, nil
	case OpAndNot:
		return token.AND_NOT_ASSIGN, nil
	default:
		return 0, fmt.Errorf("%w: assign operator", errSyntax)
	}
}

func opFromAssign(tok token.Token) (Op, bool, error) {
	switch tok {
	case token.ASSIGN:
		return 0, false, nil
	case token.DEFINE:
		return 0, true, nil
	case token.ADD_ASSIGN:
		return OpAdd, false, nil
	case token.SUB_ASSIGN:
		return OpSub, false, nil
	case token.MUL_ASSIGN:
		return OpMul, false, nil
	case token.QUO_ASSIGN:
		return OpQuo, false, nil
	case token.REM_ASSIGN:
		return OpRem, false, nil
	case token.AND_ASSIGN:
		return OpAnd, false, nil
	case token.OR_ASSIGN:
		return OpOr, false, nil
	case token.XOR_ASSIGN:
		return OpXor, false, nil
	case token.SHL_ASSIGN:
		return OpShl, false, nil
	case token.SHR_ASSIGN:
		return OpShr, false, nil
	case token.AND_NOT_ASSIGN:
		return OpAndNot, false, nil
	default:
		return 0, false, fmt.Errorf("%w: assign %s", errSyntax, tok)
	}
}

func (kind LitKind) token() token.Token {
	switch kind {
	case LitInt:
		return token.INT
	case LitFloat:
		return token.FLOAT
	case LitImag:
		return token.IMAG
	case LitRune:
		return token.CHAR
	case LitString:
		return token.STRING
	default:
		return token.ILLEGAL
	}
}

func litFrom(tok token.Token) (LitKind, error) {
	switch tok {
	case token.INT:
		return LitInt, nil
	case token.FLOAT:
		return LitFloat, nil
	case token.IMAG:
		return LitImag, nil
	case token.CHAR:
		return LitRune, nil
	case token.STRING:
		return LitString, nil
	default:
		return 0, fmt.Errorf("%w: literal %s", errSyntax, tok)
	}
}

func (dir ChanDir) ast() ast.ChanDir {
	switch dir {
	case ChanSend:
		return ast.SEND
	case ChanRecv:
		return ast.RECV
	default:
		return ast.SEND | ast.RECV
	}
}

func chanFrom(dir ast.ChanDir) ChanDir {
	switch dir {
	case ast.SEND:
		return ChanSend
	case ast.RECV:
		return ChanRecv
	default:
		return ChanBoth
	}
}

func (branch Branch) token() (token.Token, error) {
	switch branch {
	case BranchBreak:
		return token.BREAK, nil
	case BranchContinue:
		return token.CONTINUE, nil
	case BranchGoto:
		return token.GOTO, nil
	case BranchFallthrough:
		return token.FALLTHROUGH, nil
	default:
		return 0, fmt.Errorf("%w: branch", errSyntax)
	}
}

func branchFrom(tok token.Token) (Branch, error) {
	switch tok {
	case token.BREAK:
		return BranchBreak, nil
	case token.CONTINUE:
		return BranchContinue, nil
	case token.GOTO:
		return BranchGoto, nil
	case token.FALLTHROUGH:
		return BranchFallthrough, nil
	default:
		return 0, fmt.Errorf("%w: branch %s", errSyntax, tok)
	}
}
