package proto

// Term is one proto token. Punctuation is a literal and has no term.
//
//lewkit:enum
type Term string

const (
	TermSyntax     Term = "syntax"
	TermPackage    Term = "package"
	TermImport     Term = "import"
	TermPublic     Term = "public"
	TermWeak       Term = "weak"
	TermOption     Term = "option"
	TermMessage    Term = "message"
	TermEnum       Term = "enum"
	TermService    Term = "service"
	TermReserved   Term = "reserved"
	TermOneof      Term = "oneof"
	TermRPC        Term = "rpc"
	TermReturns    Term = "returns"
	TermStream     Term = "stream"
	TermOptional   Term = "optional"
	TermRequired   Term = "required"
	TermRepeated   Term = "repeated"
	TermExtend     Term = "extend"
	TermExtensions Term = "extensions"
	TermGroup      Term = "group"
	TermTo         Term = "to"
	TermMax        Term = "max"
	TermIdentifier Term = "identifier"
	TermString     Term = "string"
	TermInteger    Term = "integer"
	TermFloat      Term = "float"
	TermSpace      Term = "space"
	TermComment    Term = "comment"
)

// Rule is one proto rule. Hidden rules the builder invents are not members.
//
//lewkit:enum
type Rule string

const (
	RuleFile           Rule = "file"
	RuleSyntax         Rule = "syntax_declaration"
	RulePackage        Rule = "package_declaration"
	RuleImport         Rule = "import_declaration"
	RuleOption         Rule = "option_declaration"
	RuleMessage        Rule = "message_declaration"
	RuleEnum           Rule = "enum_declaration"
	RuleService        Rule = "service_declaration"
	RuleOneof          Rule = "oneof_declaration"
	RuleField          Rule = "field_declaration"
	RuleEnumValue      Rule = "enum_value"
	RuleEnumOption     Rule = "enum_option"
	RuleEnumReserved   Rule = "enum_reserved"
	RuleRPC            Rule = "rpc_declaration"
	RuleRPCType        Rule = "rpc_type"
	RuleReserved       Rule = "reserved_declaration"
	RuleType           Rule = "type_expression"
	RuleTypeName       Rule = "type_name"
	RuleFullIdentifier Rule = "full_identifier"
	RuleConstant       Rule = "constant"
	RuleScalar         Rule = "scalar"
	RuleStrings        Rule = "strings"
	RuleAssign         Rule = "assign"
	RuleOptionName     Rule = "option_name"
	RuleBrackets       Rule = "brackets"
	RuleRange          Rule = "range_span"
)
