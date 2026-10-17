package llvm

// Kind is the shape of a type.
//
//lewkit:enum
type Kind string

const (
	KindVoid   Kind = "void"
	KindInt    Kind = "int"
	KindFloat  Kind = "float"
	KindPtr    Kind = "ptr"
	KindArray  Kind = "array"
	KindVector Kind = "vector"
	KindStruct Kind = "struct"
	KindFunc   Kind = "func"
	KindNamed  Kind = "named"
	KindMeta   Kind = "metadata"
	KindToken  Kind = "token"
	KindLabel  Kind = "label"
	KindPrim   Kind = "prim"
)

// Float is a floating type keyword.
//
//lewkit:enum
type Float string

const (
	FloatHalf   Float = "half"
	FloatFloat  Float = "float"
	FloatDouble Float = "double"
)

// Linkage is the linkage of a global or a function.
// An empty linkage is omitted, which is LLVM's default.
//
//lewkit:enum
type Linkage string

const (
	LinkageExternal            Linkage = "external"
	LinkagePrivate             Linkage = "private"
	LinkageInternal            Linkage = "internal"
	LinkageAvailableExternally Linkage = "available_externally"
	LinkageLinkonce            Linkage = "linkonce"
	LinkageWeak                Linkage = "weak"
	LinkageCommon              Linkage = "common"
	LinkageAppending           Linkage = "appending"
	LinkageExternWeak          Linkage = "extern_weak"
	LinkageLinkonceODR         Linkage = "linkonce_odr"
	LinkageWeakODR             Linkage = "weak_odr"
)

// Unnamed is unnamed_addr or local_unnamed_addr.
// An empty value omits the specifier.
//
//lewkit:enum
type Unnamed string

const (
	UnnamedAddr  Unnamed = "unnamed_addr"
	UnnamedLocal Unnamed = "local_unnamed_addr"
)

// Op is a binary instruction opcode.
//
//lewkit:enum
type Op string

const (
	OpAdd  Op = "add"
	OpFAdd Op = "fadd"
	OpSub  Op = "sub"
	OpFSub Op = "fsub"
	OpMul  Op = "mul"
	OpFMul Op = "fmul"
	OpUDiv Op = "udiv"
	OpSDiv Op = "sdiv"
	OpFDiv Op = "fdiv"
	OpURem Op = "urem"
	OpSRem Op = "srem"
	OpFRem Op = "frem"
	OpShl  Op = "shl"
	OpLShr Op = "lshr"
	OpAShr Op = "ashr"
	OpAnd  Op = "and"
	OpOr   Op = "or"
	OpXor  Op = "xor"
)

// IPred is an integer comparison predicate.
//
//lewkit:enum
type IPred string

const (
	IPredEQ  IPred = "eq"
	IPredNE  IPred = "ne"
	IPredUGT IPred = "ugt"
	IPredUGE IPred = "uge"
	IPredULT IPred = "ult"
	IPredULE IPred = "ule"
	IPredSGT IPred = "sgt"
	IPredSGE IPred = "sge"
	IPredSLT IPred = "slt"
	IPredSLE IPred = "sle"
)

// FPred is a floating comparison predicate.
//
//lewkit:enum
type FPred string

const (
	FPredFalse FPred = "false"
	FPredOEQ   FPred = "oeq"
	FPredOGT   FPred = "ogt"
	FPredOGE   FPred = "oge"
	FPredOLT   FPred = "olt"
	FPredOLE   FPred = "ole"
	FPredONE   FPred = "one"
	FPredORD   FPred = "ord"
	FPredUEQ   FPred = "ueq"
	FPredUGT   FPred = "ugt"
	FPredUGE   FPred = "uge"
	FPredULT   FPred = "ult"
	FPredULE   FPred = "ule"
	FPredUNE   FPred = "une"
	FPredUNO   FPred = "uno"
	FPredTrue  FPred = "true"
)

// Cast is a conversion opcode.
//
//lewkit:enum
type Cast string

const (
	CastTrunc     Cast = "trunc"
	CastZExt      Cast = "zext"
	CastSExt      Cast = "sext"
	CastFPTrunc   Cast = "fptrunc"
	CastFPExt     Cast = "fpext"
	CastFPToUI    Cast = "fptoui"
	CastFPToSI    Cast = "fptosi"
	CastUIToFP    Cast = "uitofp"
	CastSIToFP    Cast = "sitofp"
	CastPtrToInt  Cast = "ptrtoint"
	CastIntToPtr  Cast = "inttoptr"
	CastBitCast   Cast = "bitcast"
	CastAddrSpace Cast = "addrspacecast"
)

// Tail is a call marker. An empty value is an ordinary call.
//
//lewkit:enum
type Tail string

const (
	TailTail Tail = "tail"
	TailMust Tail = "musttail"
	TailNo   Tail = "notail"
)

// InstKind is one instruction in a basic block.
//
//lewkit:enum
type InstKind string

const (
	InstAlloca      InstKind = "alloca"
	InstLoad        InstKind = "load"
	InstStore       InstKind = "store"
	InstGEP         InstKind = "getelementptr"
	InstBin         InstKind = "bin"
	InstICmp        InstKind = "icmp"
	InstFCmp        InstKind = "fcmp"
	InstCast        InstKind = "cast"
	InstPhi         InstKind = "phi"
	InstExtract     InstKind = "extractvalue"
	InstInsert      InstKind = "insertvalue"
	InstCall        InstKind = "call"
	InstBr          InstKind = "br"
	InstRet         InstKind = "ret"
	InstUnreachable InstKind = "unreachable"
	InstFNeg        InstKind = "fneg"
	InstFreeze      InstKind = "freeze"
	InstSelect      InstKind = "select"
	InstExtractElt  InstKind = "extractelement"
	InstInsertElt   InstKind = "insertelement"
	InstShuffle     InstKind = "shufflevector"
	InstAtomicRMW   InstKind = "atomicrmw"
	InstCmpXchg     InstKind = "cmpxchg"
	InstFence       InstKind = "fence"
	InstSwitch      InstKind = "switch"
	InstInvoke      InstKind = "invoke"
	InstCallBr      InstKind = "callbr"
	InstResume      InstKind = "resume"
	InstLanding     InstKind = "landingpad"
	InstIndirectBr  InstKind = "indirectbr"
	InstVAArg       InstKind = "va_arg"
)

// ValueKind is the form of an operand.
//
//lewkit:enum
type ValueKind string

const (
	ValLocal  ValueKind = "local"
	ValGlobal ValueKind = "global"
	ValInt    ValueKind = "int"
	ValFloat  ValueKind = "float"
	ValNull   ValueKind = "null"
	ValZero   ValueKind = "zeroinitializer"
	ValUndef  ValueKind = "undef"
	ValPoison ValueKind = "poison"
	ValTrue   ValueKind = "true"
	ValFalse  ValueKind = "false"
	ValArray  ValueKind = "array"
	ValStruct ValueKind = "struct"
	ValPacked ValueKind = "packed"
	ValVector ValueKind = "vector"
	ValString ValueKind = "cstring"
	ValGEP    ValueKind = "getelementptr"
	ValExpr   ValueKind = "expr"
	ValMeta   ValueKind = "meta"
)

// ComdatKind is the selection kind of a comdat.
//
//lewkit:enum
type ComdatKind string

const (
	ComdatAny     ComdatKind = "any"
	ComdatExact   ComdatKind = "exactmatch"
	ComdatLargest ComdatKind = "largest"
	ComdatNoDup   ComdatKind = "nodeduplicate"
	ComdatSame    ComdatKind = "samesize"
)
