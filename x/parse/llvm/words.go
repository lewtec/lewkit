package llvm

// keywordList is every literal the scanner may emit for a bare word.
// The grammar registers the same words, so a keyword is never an ident.
var keywordList = []string{
	"source_filename", "target", "datalayout", "triple", "module", "asm",
	"define", "declare", "attributes", "type", "opaque", "comdat",
	"global", "constant", "alias", "ifunc", "distinct",
	"external", "private", "internal", "available_externally", "linkonce",
	"weak", "common", "appending", "extern_weak", "linkonce_odr", "weak_odr",
	"dso_local", "unnamed_addr", "local_unnamed_addr",
	"void", "half", "float", "double", "ptr", "metadata", "token", "label",
	"bfloat", "x86_fp80", "fp128", "ppc_fp128", "x86_mmx", "x86_amx", "addrspace",
	"true", "false", "null", "undef", "poison", "zeroinitializer",
	"alloca", "load", "store", "getelementptr", "icmp", "fcmp", "phi",
	"extractvalue", "insertvalue", "call", "br", "ret", "unreachable",
	"fneg", "freeze", "select", "extractelement", "insertelement", "shufflevector",
	"atomicrmw", "cmpxchg", "fence", "switch", "invoke", "callbr", "resume",
	"landingpad", "indirectbr", "va_arg", "tail", "musttail", "notail",
	"trunc", "zext", "sext", "fptrunc", "fpext", "fptoui", "fptosi", "uitofp",
	"sitofp", "ptrtoint", "inttoptr", "bitcast", "addrspacecast",
	"add", "fadd", "sub", "fsub", "mul", "fmul", "udiv", "sdiv", "fdiv",
	"urem", "srem", "frem", "shl", "lshr", "ashr", "and", "or", "xor",
	"eq", "ne", "ugt", "uge", "ult", "ule", "sgt", "sge", "slt", "sle",
	"oeq", "ogt", "oge", "olt", "ole", "one", "ord", "ueq", "une", "uno",
	"nuw", "nsw", "exact", "disjoint", "inbounds", "nusw", "inrange",
	"volatile", "atomic", "inalloca", "nneg",
	"nnan", "ninf", "nsz", "arcp", "contract", "afn", "reassoc", "fast",
	"unordered", "monotonic", "acquire", "release", "acq_rel", "seq_cst",
	"xchg", "nand", "max", "min", "umax", "umin", "fmax", "fmin",
	"uinc_wrap", "udec_wrap", "usub_cond", "usub_sat",
	"to", "unwind", "cleanup", "catch", "filter", "syncscope",
	"personality", "prefix", "prologue", "section", "partition", "gc",
	"align", "alignstack",
	"ccc", "fastcc", "coldcc", "ghccc", "cc", "webkit_jscc", "anyregcc",
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

// keywords is the scanner's closed set. The grammar lits the same words.
var keywords = map[string]struct{}{}

func init() {
	for _, word := range keywordList {
		keywords[word] = struct{}{}
	}
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

func isAttr(name string) bool {
	switch name {
	case "noundef", "nonnull", "noalias", "nocapture", "readonly", "readnone", "writeonly",
		"signext", "zeroext", "inreg", "returned", "align", "alignstack", "nneg", "nofpclass",
		"dereferenceable", "dereferenceable_or_null", "sret", "byval", "byref", "preallocated",
		"inalloca", "elementtype", "immarg", "nofree", "nest", "swiftself", "swifterror",
		"swiftasync", "captures", "writable", "dead_on_unwind", "dead_on_return", "range",
		"allocalign", "allocptr", "allocsize", "initializes":
		return true
	default:
		return false
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

func intBits(text string) (int, bool) {
	if len(text) < 2 || text[0] != 'i' {
		return 0, false
	}
	for _, c := range text[1:] {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n := 0
	for _, c := range text[1:] {
		n = n*10 + int(c-'0')
		if n < 0 {
			return 0, false
		}
	}
	if n <= 0 {
		return 0, false
	}
	return n, true
}

func isFloatText(text string) bool {
	if len(text) == 0 {
		return false
	}
	for i := 0; i < len(text); i++ {
		switch text[i] {
		case '.', 'p', 'P', 'e', 'E', 'x', 'X':
			return true
		}
	}
	return false
}
