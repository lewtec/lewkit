package llvm

// Classifiers copied for lower and print. The scanner copies live in x/parse/llvm.

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentStart(c byte) bool {
	return isLetter(c) || c == '$' || c == '_'
}

func isSigilStart(c byte) bool {
	return isIdentStart(c) || c == '.' || c == '-'
}

func isSigilCont(c byte) bool {
	return isSigilStart(c) || isDigit(c)
}

func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
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
