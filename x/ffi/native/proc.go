package native

// Proc is one symbol from a library opened through OpenChain.
// The library and the symbol are loaded on the first Find or Call.
type Proc struct {
	load func() (uintptr, error)
}

// ProcOf returns a procedure from lib. lib is a bare soname or a path.
func ProcOf(lib, name string) Proc {
	return Proc{load: Singleton(func() (uintptr, error) {
		handle, err := OpenChain(0, lib)
		if err != nil {
			return 0, err
		}
		return Symbol(handle, name)
	})}
}

// Find loads the procedure.
func (p Proc) Find() error {
	_, err := p.load()
	return err
}
