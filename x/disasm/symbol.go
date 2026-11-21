package disasm

// Symbol is a named address from an object file.
type Symbol struct {
	Name    string
	Address uint64
}

// Symbols is a symbol table. Function symbols are listed first so
// [Symbols.Lookup] prefers them when two names share an address.
type Symbols []Symbol

// Lookup maps address to name. The first name at each address wins.
func (symbols Symbols) Lookup() map[uint64]string {
	out := make(map[uint64]string, len(symbols))
	for _, symbol := range symbols {
		if symbol.Name == "" {
			continue
		}
		if _, exists := out[symbol.Address]; exists {
			continue
		}
		out[symbol.Address] = symbol.Name
	}
	return out
}
