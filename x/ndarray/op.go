package ndarray

// DType is a kernel element type.
type DType uint8

const (
	_ DType = iota
	F32
	I32
	U8
)

func (d DType) String() string {
	switch d {
	case F32:
		return "f32"
	case I32:
		return "i32"
	case U8:
		return "u8"
	default:
		return "dtype"
	}
}

func (d DType) glsl() string {
	switch d {
	case I32:
		return "int"
	case U8:
		return "uint"
	default:
		return "float"
	}
}

func (d DType) size() int {
	switch d {
	case U8:
		return 1
	case F32, I32:
		return 4
	default:
		return 0
	}
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
	MultiplyAccumulate
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
	case WHERE, MultiplyAccumulate:
		return 3
	default:
		return -1
	}
}

var opName = [...]string{
	EXP2: "EXP2", LOG2: "LOG2", CAST: "CAST", SIN: "SIN", SQRT: "SQRT", RECIP: "RECIP", NEG: "NEG",
	ADD: "ADD", MUL: "MUL", IDIV: "IDIV", MAX: "MAX", MOD: "MOD",
	CMPLT: "CMPLT", CMPNE: "CMPNE", XOR: "XOR", SHL: "SHL", SHR: "SHR", OR: "OR", AND: "AND",
	WHERE: "WHERE", MultiplyAccumulate: "MultiplyAccumulate",
}
