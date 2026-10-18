package proto

import syn "github.com/lewtec/lewkit/x/parse/proto"

// Term and Rule are the proto tokens and rules. The grammar lives in x/parse/proto.
type (
	Term = syn.Term
	Rule = syn.Rule
)

const (
	TermSyntax         = syn.TermSyntax
	TermPackage        = syn.TermPackage
	TermImport         = syn.TermImport
	TermPublic         = syn.TermPublic
	TermWeak           = syn.TermWeak
	TermOption         = syn.TermOption
	TermMessage        = syn.TermMessage
	TermEnum           = syn.TermEnum
	TermService        = syn.TermService
	TermReserved       = syn.TermReserved
	TermOneof          = syn.TermOneof
	TermRPC            = syn.TermRPC
	TermReturns        = syn.TermReturns
	TermStream         = syn.TermStream
	TermOptional       = syn.TermOptional
	TermRequired       = syn.TermRequired
	TermRepeated       = syn.TermRepeated
	TermExtend         = syn.TermExtend
	TermExtensions     = syn.TermExtensions
	TermGroup          = syn.TermGroup
	TermTo             = syn.TermTo
	TermMax            = syn.TermMax
	TermIdentifier     = syn.TermIdentifier
	TermString         = syn.TermString
	TermInteger        = syn.TermInteger
	TermFloat          = syn.TermFloat
	TermSpace          = syn.TermSpace
	TermComment        = syn.TermComment
	RuleFile           = syn.RuleFile
	RuleSyntax         = syn.RuleSyntax
	RulePackage        = syn.RulePackage
	RuleImport         = syn.RuleImport
	RuleOption         = syn.RuleOption
	RuleMessage        = syn.RuleMessage
	RuleEnum           = syn.RuleEnum
	RuleService        = syn.RuleService
	RuleOneof          = syn.RuleOneof
	RuleField          = syn.RuleField
	RuleEnumValue      = syn.RuleEnumValue
	RuleEnumOption     = syn.RuleEnumOption
	RuleEnumReserved   = syn.RuleEnumReserved
	RuleRPC            = syn.RuleRPC
	RuleRPCType        = syn.RuleRPCType
	RuleReserved       = syn.RuleReserved
	RuleType           = syn.RuleType
	RuleTypeName       = syn.RuleTypeName
	RuleFullIdentifier = syn.RuleFullIdentifier
	RuleConstant       = syn.RuleConstant
	RuleScalar         = syn.RuleScalar
	RuleStrings        = syn.RuleStrings
	RuleAssign         = syn.RuleAssign
	RuleOptionName     = syn.RuleOptionName
	RuleBrackets       = syn.RuleBrackets
	RuleRange          = syn.RuleRange
)
