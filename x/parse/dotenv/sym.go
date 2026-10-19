package dotenv

// Term is one dotenv token.
//
//lewkit:enum
type Term string

const (
	TermExport        Term = "export"
	TermKey           Term = "key"
	TermEqual         Term = "="
	TermSpace         Term = "space"
	TermNewline       Term = "newline"
	TermComment       Term = "comment"
	TermInlineComment Term = "inline_comment"
	TermBare          Term = "bare"
	TermHash          Term = "hash"
	TermDoubleQuote   Term = "double_quote"
	TermSingleQuote   Term = "single_quote"
)

// Rule is one dotenv rule. Hidden rules the builder invents are not members.
//
//lewkit:enum
type Rule string

const (
	RuleFile   Rule = "file"
	RuleAssign Rule = "assign"
	RuleNote   Rule = "note"
	RuleBlank  Rule = "blank"
	RuleTail   Rule = "tail"
)
