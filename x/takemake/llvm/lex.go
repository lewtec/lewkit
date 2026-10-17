package llvm

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isIdentStart(c byte) bool {
	return isLetter(c) || c == '$' || c == '_'
}

func isIdentCont(c byte) bool {
	return isIdentStart(c) || isDigit(c) || c == '.' || c == '-'
}

func isSigilStart(c byte) bool {
	return isIdentStart(c) || c == '.' || c == '-'
}

func isSigilCont(c byte) bool {
	return isSigilStart(c) || isDigit(c)
}

func isHex(c byte) bool {
	return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
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
