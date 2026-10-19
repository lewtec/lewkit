package golang

import (
	"go/scanner"
	"go/token"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/parse"
)

// goLex is the Go scanner for one Parse call.
type goLex struct {
	sc   scanner.Scanner
	file *token.File
	err  error
	off  int
}

func scan(src []byte, _ int, _ func(parse.SymbolID) bool, st *parse.LexState) (parse.SymbolID, int, int, bool) {
	gs := lexer(st, src)
	if gs.err != nil {
		return 0, gs.off, gs.off, false
	}
	pos, tok, _ := gs.sc.Scan()
	if gs.err != nil {
		gs.off = gs.file.Offset(pos)
		return 0, gs.off, gs.off, false
	}
	end := gs.file.Offset(gs.sc.End())
	if tok == token.EOF {
		off := gs.file.Offset(pos)
		return parse.SymEOF, off, off, true
	}
	if tok == token.ILLEGAL {
		off := gs.file.Offset(pos)
		return 0, off, end, false
	}
	id, ok := st.Symbol(tok.String())
	if !ok {
		off := gs.file.Offset(pos)
		return 0, off, end, false
	}
	return id, gs.file.Offset(pos), end, true
}

func lexer(st *parse.LexState, src []byte) *goLex {
	if st.Data != nil {
		return st.Data.(*goLex)
	}
	gs := &goLex{}
	fset := token.NewFileSet()
	gs.file = fset.AddFile("", fset.Base(), len(src))
	gs.sc.Init(gs.file, src, func(pos token.Position, msg string) {
		if gs.err == nil {
			gs.off = pos.Offset
			gs.err = errs.Wrap(errs.ErrSyntax, msg)
		}
	}, 0)
	st.Data = gs
	return gs
}
