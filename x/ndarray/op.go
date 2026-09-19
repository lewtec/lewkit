package ndarray

// DType is a kernel element type.
type DType uint8

const (
	_ DType = iota
	F32
	I32
)

func (d DType) String() string {
	switch d {
	case F32:
		return "f32"
	case I32:
		return "i32"
	default:
		return "dtype"
	}
}

func (d DType) glsl() string {
	if d == I32 {
		return "int"
	}
	return "float"
}

// Op is one of the 21 ALU ops.
type Op uint8

const (
	_ Op = iota
	EXP2
	LOG2
	CAST
	SIN
	SQRT
	RECIP
	NEG
	ADD
	MUL
	IDIV
	MAX
	MOD
	CMPLT
	CMPNE
	XOR
	SHL
	SHR
	OR
	AND
	WHERE
	MULACC
)

func (op Op) String() string {
	if int(op) < len(opName) {
		return opName[op]
	}
	return "op"
}

func (op Op) arity() int {
	switch op {
	case EXP2, LOG2, CAST, SIN, SQRT, RECIP, NEG:
		return 1
	case ADD, MUL, IDIV, MAX, MOD, CMPLT, CMPNE, XOR, SHL, SHR, OR, AND:
		return 2
	case WHERE, MULACC:
		return 3
	default:
		return -1
	}
}

var opName = [...]string{
	EXP2: "EXP2", LOG2: "LOG2", CAST: "CAST", SIN: "SIN", SQRT: "SQRT", RECIP: "RECIP", NEG: "NEG",
	ADD: "ADD", MUL: "MUL", IDIV: "IDIV", MAX: "MAX", MOD: "MOD",
	CMPLT: "CMPLT", CMPNE: "CMPNE", XOR: "XOR", SHL: "SHL", SHR: "SHR", OR: "OR", AND: "AND",
	WHERE: "WHERE", MULACC: "MULACC",
}
