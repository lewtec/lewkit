package llvm

import syn "github.com/lewtec/lewkit/x/parse/llvm"

type (
	Term = syn.Term
	Rule = syn.Rule
)

const (
	TermIdentifier              = syn.TermIdentifier
	TermNumber                  = syn.TermNumber
	TermString                  = syn.TermString
	TermCString                 = syn.TermCString
	TermLocalReference          = syn.TermLocalReference
	TermGlobalReference         = syn.TermGlobalReference
	TermLocalDefinition         = syn.TermLocalDefinition
	TermGlobalDefinition        = syn.TermGlobalDefinition
	TermIntegerType             = syn.TermIntegerType
	TermComdatName              = syn.TermComdatName
	TermLabelName               = syn.TermLabelName
	TermExclamation             = syn.TermExclamation
	TermMetadataExclamation     = syn.TermMetadataExclamation
	TermAttributeGroupReference = syn.TermAttributeGroupReference
	TermBraced                  = syn.TermBraced
	TermEllipsis                = syn.TermEllipsis
	RuleFile                    = syn.RuleFile
	RuleSource                  = syn.RuleSource
	RuleLayout                  = syn.RuleLayout
	RuleTripleDefinition        = syn.RuleTripleDefinition
	RuleModuleAssembly          = syn.RuleModuleAssembly
	RuleTypedef                 = syn.RuleTypedef
	RuleGlobalVariable          = syn.RuleGlobalVariable
	RuleFunction                = syn.RuleFunction
	RuleHeader                  = syn.RuleHeader
	RuleParameters              = syn.RuleParameters
	RuleParameter               = syn.RuleParameter
	RuleBody                    = syn.RuleBody
	RuleInstruction             = syn.RuleInstruction
	RuleType                    = syn.RuleType
	RuleTypeWithoutStruct       = syn.RuleTypeWithoutStruct
	RuleStar                    = syn.RuleStar
	RuleFunctionTail            = syn.RuleFunctionTail
	RuleAddressSpace            = syn.RuleAddressSpace
	RuleWord                    = syn.RuleWord
	RuleAlignment               = syn.RuleAlignment
	RuleValue                   = syn.RuleValue
	RuleTypedValue              = syn.RuleTypedValue
	RuleOperand                 = syn.RuleOperand
	RuleOperandWithoutStruct    = syn.RuleOperandWithoutStruct
	RuleGetElementPointer       = syn.RuleGetElementPointer
	RuleConstantExpression      = syn.RuleConstantExpression
	RuleAggregate               = syn.RuleAggregate
	RuleMetadataNode            = syn.RuleMetadataNode
	RuleAttach                  = syn.RuleAttach
	RulePiece                   = syn.RulePiece
	RuleSync                    = syn.RuleSync
	RuleCallingConventionNumber = syn.RuleCallingConventionNumber
	RuleCallBody                = syn.RuleCallBody
	RuleTopMetadata             = syn.RuleTopMetadata
	RuleAttributeGroup          = syn.RuleAttributeGroup
	RuleComdatDefinition        = syn.RuleComdatDefinition
)
