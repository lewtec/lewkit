package leaven

import (
	"runtime"
	"unsafe"

	"github.com/lewtec/leaven-tree-sitter/grammar"
)

//go:linkname tsParserNew github.com/lewtec/leaven-tree-sitter/grammar.ts_parser_new
func tsParserNew() *grammar.TSParser

//go:linkname tsParserDelete github.com/lewtec/leaven-tree-sitter/grammar.ts_parser_delete
func tsParserDelete(parser *grammar.TSParser)

//go:linkname tsParserSetLanguage github.com/lewtec/leaven-tree-sitter/grammar.ts_parser_set_language
func tsParserSetLanguage(parser *grammar.TSParser, language *grammar.TSLanguage) bool

//go:linkname tsParserParse github.com/lewtec/leaven-tree-sitter/grammar.ts_parser_parse
func tsParserParse(parser *grammar.TSParser, old *grammar.TSTree, input *grammar.TSInput) *grammar.TSTree

//go:linkname tsStringInputRead github.com/lewtec/leaven-tree-sitter/grammar.ts_string_input_read
func tsStringInputRead(self *byte, byteIndex int32, point int64, length *int32) *byte

//go:linkname wrapTree github.com/lewtec/leaven-tree-sitter/grammar.wrapTree
func wrapTree(lang grammar.Language, native *grammar.TSTree) *grammar.Tree

// pinnedParser parses with buffers this package owns for the whole call.
// ParseBytes on the grammar parser stores the source in a record that
// only the calloc'd parser points at, so the collector can free it mid-lex.
type pinnedParser struct {
	native *grammar.TSParser
	lang   grammar.Language
}

func newPinnedParser() *pinnedParser {
	return &pinnedParser{native: tsParserNew()}
}

func (p *pinnedParser) Delete() {
	if p == nil || p.native == nil {
		return
	}
	tsParserDelete(p.native)
	p.native = nil
}

func (p *pinnedParser) SetLanguage(lang grammar.Language) bool {
	if p == nil || p.native == nil {
		return false
	}
	if !tsParserSetLanguage(p.native, lang) {
		return false
	}
	p.lang = lang
	return true
}

func (p *pinnedParser) ParseBytes(source []byte) *grammar.Tree {
	if p == nil || p.native == nil {
		return nil
	}
	native := parsePinned(p.native, source)
	if native == nil {
		return nil
	}
	return wrapTree(p.lang, native)
}

// parsePinned keeps the NUL-terminated source and the string-input record
// alive and unmoved until the parse returns. The lexer copies both pointers
// into the calloc'd parser, which the collector does not scan.
func parsePinned(parser *grammar.TSParser, source []byte) *grammar.TSTree {
	terminated := make([]byte, len(source)+1)
	copy(terminated, source)
	stringInput := &grammar.TSStringInput{
		F0: &terminated[0],
		F1: int32(len(source)),
	}
	parserInput := &grammar.TSInput{
		F0: (*byte)(unsafe.Pointer(stringInput)),
		F1: tsStringInputRead,
	}
	var pin runtime.Pinner
	pin.Pin(&terminated[0])
	pin.Pin(stringInput)
	defer pin.Unpin()
	tree := tsParserParse(parser, nil, parserInput)
	runtime.KeepAlive(terminated)
	runtime.KeepAlive(stringInput)
	runtime.KeepAlive(parserInput)
	return tree
}
