package dotenv

import syntax "github.com/lewtec/lewkit/x/parse/dotenv"

// Term, Rule, and Quote are the dotenv grammar. It lives in x/parse/dotenv.
type (
	Term  = syntax.Term
	Rule  = syntax.Rule
	Quote = syntax.Quote
)

const (
	TermExport        = syntax.TermExport
	TermKey           = syntax.TermKey
	TermEqual         = syntax.TermEqual
	TermSpace         = syntax.TermSpace
	TermNewline       = syntax.TermNewline
	TermComment       = syntax.TermComment
	TermInlineComment = syntax.TermInlineComment
	TermBare          = syntax.TermBare
	TermHash          = syntax.TermHash
	TermDoubleQuote   = syntax.TermDoubleQuote
	TermSingleQuote   = syntax.TermSingleQuote
	RuleFile          = syntax.RuleFile
	RuleAssign        = syntax.RuleAssign
	RuleNote          = syntax.RuleNote
	RuleBlank         = syntax.RuleBlank
	RuleTail          = syntax.RuleTail
	QuoteBare         = syntax.QuoteBare
	QuoteSingle       = syntax.QuoteSingle
	QuoteDouble       = syntax.QuoteDouble
)

var ErrQuote = syntax.ErrQuote
